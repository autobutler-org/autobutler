-- Photo perceptual hash index for near-duplicate detection.
--
-- device_serial + rel_path identify a file uniquely (same as photo_favorites).
-- dhash is the 64-bit difference hash computed from the image as a hex string.
-- content_hash is the SHA-256 of the raw file bytes for exact-duplicate detection.
-- Both are nullable — a row may be inserted with only one hash when the other
-- is not yet available (e.g. content_hash is free but dhash requires decoding).
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
