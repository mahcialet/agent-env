# Database schema

Generated from SQL migrations. Do not edit; run `go run ./tools/repoctl generate`.

## `migrations/001_initial.sql`

```sql
CREATE TABLE repositories (
  id TEXT PRIMARY KEY,
  canonical_path TEXT NOT NULL
);
CREATE TABLE leases (
  id TEXT PRIMARY KEY,
  owner TEXT NOT NULL,
  purpose TEXT NOT NULL,
  mode TEXT NOT NULL,
  repository TEXT NOT NULL,
  stack TEXT NOT NULL,
  desired_state TEXT NOT NULL,
  observed_state TEXT NOT NULL,
  created_at TEXT NOT NULL,
  heartbeat_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  manifest_digest TEXT NOT NULL,
  source_set_digest TEXT NOT NULL,
  payload TEXT NOT NULL
);
CREATE INDEX leases_active ON leases(observed_state);
CREATE INDEX leases_owner ON leases(owner);
CREATE INDEX leases_expiration ON leases(expires_at);
CREATE INDEX leases_repository ON leases(repository);
CREATE TABLE lease_sources (
  lease_id TEXT NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
  alias TEXT NOT NULL,
  repository_id TEXT NOT NULL REFERENCES repositories(id),
  requested_ref TEXT NOT NULL,
  resolved_commit TEXT NOT NULL,
  worktree_path TEXT NOT NULL,
  checkout_mode TEXT NOT NULL,
  writable INTEGER NOT NULL,
  resolved_at TEXT NOT NULL,
  PRIMARY KEY(lease_id, alias)
);
CREATE UNIQUE INDEX source_worktree_identity ON lease_sources(worktree_path) WHERE worktree_path <> '';
CREATE TABLE lease_components (
  lease_id TEXT NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  runtime TEXT NOT NULL,
  services TEXT NOT NULL,
  capabilities TEXT NOT NULL,
  resolution_order INTEGER NOT NULL,
  PRIMARY KEY(lease_id, name)
);
CREATE TABLE lease_runtimes (
  lease_id TEXT NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  project TEXT NOT NULL,
  context TEXT NOT NULL,
  active INTEGER NOT NULL,
  payload TEXT NOT NULL,
  PRIMARY KEY(lease_id, name)
);
CREATE UNIQUE INDEX runtime_project_reservation ON lease_runtimes(context, project) WHERE active = 1 AND project <> '';
CREATE TABLE runtime_resources (
  id TEXT PRIMARY KEY,
  lease_id TEXT NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
  runtime TEXT NOT NULL,
  kind TEXT NOT NULL,
  external_id TEXT NOT NULL,
  metadata TEXT NOT NULL
);
CREATE INDEX resources_lease ON runtime_resources(lease_id);
CREATE TABLE events (
  id TEXT PRIMARY KEY,
  lease_id TEXT NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
  time TEXT NOT NULL,
  type TEXT NOT NULL,
  message TEXT NOT NULL,
  payload TEXT NOT NULL
);
CREATE INDEX events_lease ON events(lease_id, time, id);
CREATE TABLE command_runs (
  id TEXT PRIMARY KEY,
  lease_id TEXT NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  source_alias TEXT NOT NULL,
  working_directory TEXT NOT NULL,
  argv_json TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL,
  exit_code INTEGER NOT NULL,
  stdout_path TEXT NOT NULL,
  stderr_path TEXT NOT NULL,
  status TEXT NOT NULL,
  payload TEXT NOT NULL,
  UNIQUE(id, lease_id)
);
CREATE INDEX runs_lease ON command_runs(lease_id, started_at, id);
CREATE TABLE artifacts (
  id TEXT PRIMARY KEY,
  lease_id TEXT NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
  run_id TEXT,
  kind TEXT NOT NULL,
  path TEXT NOT NULL,
  digest TEXT NOT NULL,
  created_at TEXT NOT NULL,
  payload TEXT NOT NULL,
  FOREIGN KEY(run_id, lease_id) REFERENCES command_runs(id, lease_id)
);
CREATE INDEX artifacts_lease ON artifacts(lease_id, created_at, id);
CREATE TABLE operation_locks (
  lease_id TEXT PRIMARY KEY REFERENCES leases(id) ON DELETE CASCADE,
  token TEXT NOT NULL,
  expires_at INTEGER NOT NULL
);
```

## `migrations/002_command_cancellation.sql`

```sql
CREATE TABLE command_run_cancellations (
  run_id TEXT PRIMARY KEY REFERENCES command_runs(id) ON DELETE CASCADE,
  requested_at TEXT NOT NULL
);
```

## `migrations/003_android_reservations.sql`

```sql
CREATE TABLE android_reservations (
  lease_id TEXT NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
  runtime_name TEXT NOT NULL,
  avd_name TEXT NOT NULL,
  avd_home TEXT NOT NULL,
  avd_path TEXT NOT NULL,
  console_port INTEGER NOT NULL CHECK(console_port BETWEEN 5554 AND 5682 AND console_port % 2 = 0),
  adb_port INTEGER NOT NULL CHECK(adb_port = console_port + 1),
  serial TEXT NOT NULL CHECK(serial = 'emulator-' || console_port),
  active INTEGER NOT NULL CHECK(active IN (0, 1)),
  PRIMARY KEY(lease_id, runtime_name)
);
CREATE UNIQUE INDEX android_console_reservation ON android_reservations(console_port) WHERE active = 1;
CREATE UNIQUE INDEX android_adb_reservation ON android_reservations(adb_port) WHERE active = 1;
CREATE UNIQUE INDEX android_avd_name_reservation ON android_reservations(avd_name) WHERE active = 1;
CREATE UNIQUE INDEX android_avd_path_reservation ON android_reservations(avd_path) WHERE active = 1;
```

## `migrations/004_process_reservations.sql`

```sql
CREATE TABLE persistent_processes (
 lease_id TEXT NOT NULL REFERENCES leases(id),
 runtime_name TEXT NOT NULL,
 snapshot_json BLOB NOT NULL,
 PRIMARY KEY (lease_id, runtime_name)
);
CREATE TABLE process_port_reservations (
 lease_id TEXT NOT NULL REFERENCES leases(id),
 runtime_name TEXT NOT NULL,
 port_name TEXT NOT NULL,
 port INTEGER NOT NULL UNIQUE CHECK (port > 0 AND port < 65536),
 PRIMARY KEY (lease_id, runtime_name, port_name)
);
```
