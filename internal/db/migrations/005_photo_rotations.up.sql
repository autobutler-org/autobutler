-- Stores per-photo rotation (0/1/2/3 × 90° CW) on the server so it
-- persists across devices and app reinstalls.
CREATE TABLE
    IF NOT EXISTS photo_rotations (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        device_serial TEXT NOT NULL DEFAULT '',
        rel_path TEXT NOT NULL,
        rotation_quarters INTEGER NOT NULL DEFAULT 0,
        updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
        UNIQUE (device_serial, rel_path)
    );
