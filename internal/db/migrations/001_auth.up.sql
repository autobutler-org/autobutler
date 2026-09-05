-- Single-user authentication: the account and the sessions issued to it.
--
-- Sessions store only the SHA-256 digest of each token, never the raw token
-- (#1334), and every timestamp is written in SQLite's canonical UTC format
-- because connections carry _time_format=datetime&_timezone=UTC (#1650). Both
-- of those were retrofits that had to delete the rows they could not convert;
-- on a fresh database there is nothing to delete, so only the schema remains.

CREATE TABLE
    IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT NOT NULL UNIQUE,
        password_hash TEXT NOT NULL,
        recovery_phrase_hash TEXT NOT NULL,
        created_at DATETIME NOT NULL DEFAULT (datetime('now')),
        -- The first user created during setup is promoted to admin by the
        -- setup handler.
        is_admin INTEGER NOT NULL DEFAULT 0
    );

CREATE TABLE
    IF NOT EXISTS sessions (
        token TEXT PRIMARY KEY,
        user_id INTEGER NOT NULL,
        expires_at DATETIME NOT NULL,
        created_at DATETIME NOT NULL DEFAULT (datetime('now')),
        -- last_used_at is what makes renewal-on-use possible, and what
        -- debounces it: ValidateSession only rewrites expires_at once this
        -- column is stale enough, rather than on every authenticated request
        -- (#1647). Without it expires_at was stamped once at login and never
        -- written again, so a user who opened the app daily was still forced
        -- back to the login screen on day 31.
        last_used_at DATETIME NOT NULL DEFAULT '1970-01-01 00:00:00',
        FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
    );
