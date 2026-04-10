-- Photo albums are virtual collections of pointers to photo files.
-- Albums can be nested arbitrarily (parent_id = NULL means root album).
CREATE TABLE
    IF NOT EXISTS photo_albums (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        parent_id INTEGER,
        created_at DATETIME NOT NULL DEFAULT (datetime('now')),
        updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
        FOREIGN KEY (parent_id) REFERENCES photo_albums (id) ON DELETE CASCADE
    );

-- Each row is a pointer: album -> photo file on disk (identified by device
-- serial + relative path within that device's Cirrus directory).
-- A photo can appear in multiple albums (copy semantics). The file itself
-- lives on disk; this table is purely metadata.
CREATE TABLE
    IF NOT EXISTS photo_album_items (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        album_id INTEGER NOT NULL,
        device_serial TEXT NOT NULL,
        rel_path TEXT NOT NULL,
        added_at DATETIME NOT NULL DEFAULT (datetime('now')),
        FOREIGN KEY (album_id) REFERENCES photo_albums (id) ON DELETE CASCADE,
        UNIQUE (album_id, device_serial, rel_path)
    );
