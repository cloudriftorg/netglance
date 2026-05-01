CREATE TABLE IF NOT EXISTS users (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  username      TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  created_at    INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
  token      TEXT PRIMARY KEY,
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS hosts (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  mac            TEXT UNIQUE NOT NULL,
  ip             TEXT NOT NULL,
  vlan_id        INTEGER,
  network_name   TEXT,
  hostname       TEXT,
  vendor         TEXT,
  custom_name    TEXT,
  first_seen     INTEGER NOT NULL,
  last_seen      INTEGER NOT NULL,
  online         INTEGER NOT NULL DEFAULT 0,
  missed_scans   INTEGER NOT NULL DEFAULT 0,
  notify_offline INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_hosts_vlan   ON hosts(vlan_id);
CREATE INDEX IF NOT EXISTS idx_hosts_online ON hosts(online);

CREATE TABLE IF NOT EXISTS host_events (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  host_id INTEGER NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
  ts      INTEGER NOT NULL,
  kind    TEXT NOT NULL,
  ip      TEXT
);
CREATE INDEX IF NOT EXISTS idx_events_host_ts ON host_events(host_id, ts DESC);

CREATE TABLE IF NOT EXISTS settings (
  k TEXT PRIMARY KEY,
  v TEXT NOT NULL
);
