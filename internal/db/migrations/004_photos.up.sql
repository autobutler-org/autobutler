-- Photo metadata. The files themselves live on disk; everything here is a
-- pointer to one, identified by device serial plus the path relative to that
-- device's files directory.

-- Photo albums are virtual collections of pointers to photo files.
-- Albums can be nested arbitrarily (parent_id = NULL means root album).
CREATE TABLE
    IF NOT EXISTS photo_albums (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        parent_id INTEGER,
        created_at DATETIME NOT NULL DEFAULT (datetime('now')),
        updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
        -- Marks a system album (favorites, inbox) rather than a user one.
        smart_type TEXT,
        FOREIGN KEY (parent_id) REFERENCES photo_albums (id) ON DELETE CASCADE
    );

-- Enforce at most one album per smart_type value (e.g. only one "favorites" album).
CREATE UNIQUE INDEX IF NOT EXISTS idx_photo_albums_smart_type
    ON photo_albums (smart_type)
    WHERE smart_type IS NOT NULL;

-- Each row is a pointer: album -> photo file on disk. A photo can appear in
-- multiple albums (copy semantics).
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

-- Per-photo rotation (0/1/2/3 x 90 degrees CW) kept on the server so it
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

-- Fast lookup for whether a photo is favorited, and the source of truth for
-- the Favorites smart album.
CREATE TABLE
    IF NOT EXISTS photo_favorites (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        device_serial TEXT NOT NULL DEFAULT '',
        rel_path TEXT NOT NULL,
        created_at DATETIME NOT NULL DEFAULT (datetime('now')),
        UNIQUE (device_serial, rel_path)
    );

-- Perceptual hash index for near-duplicate detection.
--
-- dhash is the 64-bit difference hash computed from the image as a hex string.
-- content_hash is the SHA-256 of the raw file bytes for exact-duplicate
-- detection. Both are nullable: a row may be inserted with only one hash when
-- the other is not yet available (content_hash is free, dhash requires
-- decoding the image).
CREATE TABLE IF NOT EXISTS photo_hashes (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    device_serial  TEXT NOT NULL,
    rel_path       TEXT NOT NULL,
    dhash          TEXT,      -- 16-char hex (64 bits)
    content_hash   TEXT,      -- 64-char hex (sha-256)
    computed_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(device_serial, rel_path)
);

CREATE INDEX IF NOT EXISTS idx_photo_hashes_dhash         ON photo_hashes (dhash)         WHERE dhash IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_photo_hashes_content_hash  ON photo_hashes (content_hash)  WHERE content_hash IS NOT NULL;
