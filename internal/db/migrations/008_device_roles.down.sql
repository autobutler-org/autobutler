DROP INDEX IF EXISTS idx_device_roles_one_primary;
DROP TABLE IF EXISTS device_roles;

DROP TABLE IF EXISTS device_names;
CREATE TABLE IF NOT EXISTS device_names (
    device_path TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
