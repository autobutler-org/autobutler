-- vfs_metadata: arbitrary JSON key-value pairs attached to any (namespace, path) coordinate.
-- Keys are namespaced by convention: the writing plugin's namespace ID is the key prefix
-- (e.g. "photos.rotation", "files.tag"). The host enforces prefix ownership at the API layer.
CREATE TABLE IF NOT EXISTS vfs_metadata (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    namespace   TEXT NOT NULL,
    path        TEXT NOT NULL,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,  -- JSON-encoded value
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE (namespace, path, key)
);

CREATE INDEX IF NOT EXISTS idx_vfs_metadata_ns_path
    ON vfs_metadata (namespace, path);

CREATE INDEX IF NOT EXISTS idx_vfs_metadata_ns_key
    ON vfs_metadata (namespace, key);

-- vfs_db_entries: virtual directory-style entries for DBVFS namespaces.
-- Used by the photos namespace to store album hierarchy without touching disk.
CREATE TABLE IF NOT EXISTS vfs_db_entries (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    namespace   TEXT NOT NULL,
    path        TEXT NOT NULL,   -- VFS-relative path, e.g. "/albums/summer-2024/"
    is_dir      BOOLEAN NOT NULL DEFAULT 0,
    size        INTEGER NOT NULL DEFAULT 0,
    mime_type   TEXT NOT NULL DEFAULT '',
    content     BLOB,            -- NULL for directories; raw bytes for virtual files
    created_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at  DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE (namespace, path)
);

CREATE INDEX IF NOT EXISTS idx_vfs_db_entries_ns_path
    ON vfs_db_entries (namespace, path);
