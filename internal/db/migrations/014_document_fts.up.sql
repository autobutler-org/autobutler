-- Full-text search index for document contents (#1339).
--
-- document_fts is an FTS5 virtual table. Each row represents one indexable
-- file. device_serial + rel_path identify the file; body holds the extracted
-- plain text. The content= option makes FTS5 a "content table" backed by the
-- implicit shadow tables — we manage inserts/deletes ourselves.
--
-- FTS5 requires the fts5 extension compiled into SQLite, which modernc.org/sqlite
-- includes by default.
CREATE VIRTUAL TABLE IF NOT EXISTS document_fts USING fts5(
    device_serial UNINDEXED,
    rel_path      UNINDEXED,
    body,
    tokenize      = "unicode61 remove_diacritics 1"
);
