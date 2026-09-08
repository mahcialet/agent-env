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
