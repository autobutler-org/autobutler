-- Rename device roles: primary -> default-storage, backup -> snapshot-backup.
-- SQLite cannot ALTER CHECK constraints, so we recreate the table.
CREATE TABLE device_roles_new (
    device_serial TEXT PRIMARY KEY,
    role TEXT NOT NULL DEFAULT 'unassigned'
        CHECK (role IN ('default-storage', 'snapshot-backup', 'unassigned')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO device_roles_new (device_serial, role, updated_at)
SELECT device_serial,
       CASE role
           WHEN 'primary' THEN 'default-storage'
           WHEN 'backup'  THEN 'snapshot-backup'
           ELSE role
       END,
       updated_at
FROM device_roles;

DROP TABLE device_roles;
ALTER TABLE device_roles_new RENAME TO device_roles;

CREATE UNIQUE INDEX IF NOT EXISTS idx_device_roles_one_default_storage
    ON device_roles(role) WHERE role = 'default-storage';
