-- SQLite does not support DROP COLUMN before 3.35.0; for older versions
-- a table rebuild would be required. On the Raspberry Pi target (SQLite 3.39+)
-- this is safe.
ALTER TABLE users DROP COLUMN role;
