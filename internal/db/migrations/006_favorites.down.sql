DROP TABLE IF EXISTS photo_favorites;

ALTER TABLE photo_albums DROP COLUMN retention_days;
ALTER TABLE photo_albums DROP COLUMN smart_type;
