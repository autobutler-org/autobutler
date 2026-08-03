-- Full-text search index over file contents.
--
-- The FTS5 virtual table stores extracted text for plaintext, PDF, and EPUB
-- files. It is a derived, rebuildable index — the canonical data lives in the
-- actual files on disk. Dropping and rebuilding this table is safe at any time.
--
-- device_serial + rel_path is the unique key for a file across both tables.
CREATE TABLE IF NOT EXISTS fts_documents (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    device_serial TEXT   NOT NULL DEFAULT '',
    rel_path      TEXT   NOT NULL,
    content_hash  TEXT   NOT NULL DEFAULT '', -- SHA-256 of content for incremental updates
    indexed_at    DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(device_serial, rel_path)
);

-- FTS5 virtual table. 'content' links it to fts_documents for content= mode,
-- meaning the text is stored only in fts5 (not duplicated in fts_documents).
CREATE VIRTUAL TABLE IF NOT EXISTS fts_index USING fts5(
    body,
    content=fts_documents,
    content_rowid=id,
    tokenize='porter unicode61'
);

-- Triggers to keep the FTS index consistent when rows are deleted.
CREATE TRIGGER IF NOT EXISTS fts_documents_ad AFTER DELETE ON fts_documents BEGIN
    INSERT INTO fts_index(fts_index, rowid, body) VALUES('delete', old.id, old.id);
END;
