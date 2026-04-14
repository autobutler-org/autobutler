-- photo_favorites: fast lookup for whether a photo is favorited.
-- Also serves as the source of truth for the Favorites smart album.
CREATE TABLE
    IF NOT EXISTS photo_favorites (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        device_serial TEXT NOT NULL DEFAULT '',
        rel_path TEXT NOT NULL,
        created_at DATETIME NOT NULL DEFAULT (datetime('now')),
        UNIQUE (device_serial, rel_path)
    );

-- Add smart_type and retention_days to photo_albums so we can
-- identify system albums (favorites, inbox) and configure auto-expiry.
ALTER TABLE photo_albums
ADD COLUMN smart_type TEXT;

ALTER TABLE photo_albums
ADD COLUMN retention_days INTEGER;
