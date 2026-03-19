-- Device display names: allows users to rename storage devices.
-- Keyed by device_path (e.g. "/dev/mmcblk0p2") which is stable across reboots.
CREATE TABLE IF NOT EXISTS device_names (
    device_path TEXT PRIMARY KEY,
    display_name TEXT NOT NULL,
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
