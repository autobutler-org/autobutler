CREATE TABLE device_roles_old (
    device_serial TEXT PRIMARY KEY,
    role TEXT NOT NULL DEFAULT 'unassigned'
        CHECK (role IN ('primary', 'backup', 'unassigned')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO device_roles_old (device_serial, role, updated_at)
SELECT device_serial,
       CASE role
           WHEN 'default-storage'  THEN 'primary'
           WHEN 'snapshot-backup'  THEN 'backup'
           ELSE role
       END,
       updated_at
FROM device_roles;

DROP TABLE device_roles;
ALTER TABLE device_roles_old RENAME TO device_roles;

CREATE UNIQUE INDEX IF NOT EXISTS idx_device_roles_one_primary
    ON device_roles(role) WHERE role = 'primary';
