-- Expiring public share links for files and folders.
--
-- A share link grants time-limited, read-only, unauthenticated access to a
-- specific path in the cirrus file store. Optional password protection and
-- single-use tokens are supported.
CREATE TABLE IF NOT EXISTS share_links (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    token          TEXT    NOT NULL UNIQUE,          -- URL-safe random token (32 bytes, base64url)
    device_serial  TEXT    NOT NULL DEFAULT '',
    rel_path       TEXT    NOT NULL,                 -- relative path to share (file or directory)
    password_hash  TEXT    NOT NULL DEFAULT '',      -- bcrypt hash, empty = no password
    max_uses       INTEGER NOT NULL DEFAULT 0,       -- 0 = unlimited
    use_count      INTEGER NOT NULL DEFAULT 0,
    expires_at     DATETIME,                         -- NULL = never expires
    created_at     DATETIME NOT NULL DEFAULT (datetime('now')),
    created_by     TEXT    NOT NULL DEFAULT ''       -- username who created the link
);

CREATE INDEX IF NOT EXISTS idx_share_links_token ON share_links (token);
CREATE INDEX IF NOT EXISTS idx_share_links_expires_at ON share_links (expires_at)
    WHERE expires_at IS NOT NULL;
