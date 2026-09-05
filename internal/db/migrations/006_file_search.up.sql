-- FTS5 full-text search index for file contents.
-- Stores extracted plaintext from indexed files so users can search
-- inside documents, not just by filename.
--
-- The content table stores the extracted text keyed by device serial +
-- relative path. The FTS5 virtual table mirrors it for fast full-text queries.
--
-- content=file_content makes this an "external content" FTS5 table: the FTS
-- index only stores term positions, not the original text. Snippets require
-- the content table lookup. This keeps the index compact and avoids doubling
-- storage.

CREATE TABLE IF NOT EXISTS file_content (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    serial      TEXT    NOT NULL,
    rel_path    TEXT    NOT NULL,
    extracted   TEXT    NOT NULL DEFAULT '',
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(serial, rel_path)
);

CREATE VIRTUAL TABLE IF NOT EXISTS file_content_fts USING fts5(
    extracted,
    content=file_content,
    content_rowid=id,
    tokenize='porter unicode61'
);

-- Triggers keep the FTS index in sync with file_content.
CREATE TRIGGER IF NOT EXISTS file_content_ai
AFTER INSERT ON file_content BEGIN
    INSERT INTO file_content_fts(rowid, extracted) VALUES (new.id, new.extracted);
END;

CREATE TRIGGER IF NOT EXISTS file_content_ad
AFTER DELETE ON file_content BEGIN
    INSERT INTO file_content_fts(file_content_fts, rowid, extracted) VALUES ('delete', old.id, old.extracted);
END;

CREATE TRIGGER IF NOT EXISTS file_content_au
AFTER UPDATE ON file_content BEGIN
    INSERT INTO file_content_fts(file_content_fts, rowid, extracted) VALUES ('delete', old.id, old.extracted);
    INSERT INTO file_content_fts(rowid, extracted) VALUES (new.id, new.extracted);
END;
