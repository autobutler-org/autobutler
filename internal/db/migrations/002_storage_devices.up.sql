-- Storage devices: what the user calls a drive, and what role it plays.
--
-- Both tables are keyed by device_serial (the USB serial number; the empty
-- string is the internal device). device_names was originally keyed by
-- device_path, which is not stable enough to identify a drive across
-- reconnects, and the role vocabulary was originally primary/backup.

CREATE TABLE IF NOT EXISTS device_names (
    device_serial TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS device_roles (
    device_serial TEXT PRIMARY KEY,
    role TEXT NOT NULL DEFAULT 'unassigned'
        CHECK (role IN ('default-storage', 'snapshot-backup', 'unassigned')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- At most one device may hold the 'default-storage' role.
CREATE UNIQUE INDEX IF NOT EXISTS idx_device_roles_one_default_storage
    ON device_roles(role) WHERE role = 'default-storage';
