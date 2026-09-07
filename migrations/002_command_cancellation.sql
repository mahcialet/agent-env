CREATE TABLE command_run_cancellations (
  run_id TEXT PRIMARY KEY REFERENCES command_runs(id) ON DELETE CASCADE,
  requested_at TEXT NOT NULL
);
