CREATE TABLE IF NOT EXISTS share_links (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash    TEXT    NOT NULL UNIQUE,
    created_by    INTEGER REFERENCES users(id) ON DELETE SET NULL,
    resource_type TEXT    NOT NULL CHECK(resource_type IN ('file', 'folder')),
    resource_path TEXT    NOT NULL,
    device_serial TEXT    NOT NULL DEFAULT '',
    expires_at    DATETIME,
    view_count    INTEGER NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_share_links_token_hash ON share_links (token_hash);
CREATE INDEX IF NOT EXISTS idx_share_links_expires_at ON share_links (expires_at);
CREATE INDEX IF NOT EXISTS idx_share_links_created_by ON share_links (created_by);
