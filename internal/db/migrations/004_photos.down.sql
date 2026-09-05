DROP INDEX IF EXISTS idx_photo_hashes_content_hash;

DROP INDEX IF EXISTS idx_photo_hashes_dhash;

DROP TABLE IF EXISTS photo_hashes;

DROP TABLE IF EXISTS photo_favorites;

DROP TABLE IF EXISTS photo_rotations;

DROP TABLE IF EXISTS photo_album_items;

-- The index goes before the table it sits on: SQLite refuses to drop a column
-- an index still names, and dropping the table takes the index with it either
-- way.
DROP INDEX IF EXISTS idx_photo_albums_smart_type;

DROP TABLE IF EXISTS photo_albums;
