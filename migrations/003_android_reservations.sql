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
