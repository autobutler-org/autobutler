-- Device roles: assigns a 3-2-1 backup role to storage devices.
-- Keyed by device_serial (USB serial number; empty string = internal device).
CREATE TABLE IF NOT EXISTS device_roles (
    device_serial TEXT PRIMARY KEY,
    role TEXT NOT NULL DEFAULT 'unassigned' CHECK (role IN ('primary', 'backup', 'unassigned')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- At most one device may hold the 'primary' role.
CREATE UNIQUE INDEX IF NOT EXISTS idx_device_roles_one_primary
    ON device_roles(role) WHERE role = 'primary';

-- Migrate device_names from device_path to device_serial.
-- Existing rows can't be mapped (no path→serial lookup without the device
-- connected), so we recreate the table with the new key.
DROP TABLE IF EXISTS device_names;
CREATE TABLE IF NOT EXISTS device_names (
    device_serial TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
