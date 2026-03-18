DROP TABLE IF EXISTS calendar_events;

DROP TABLE IF EXISTS calendars;

CREATE TABLE
    IF NOT EXISTS connected_devices (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        ip_address TEXT NOT NULL,
        user_agent TEXT NOT NULL DEFAULT '',
        first_seen_at DATETIME NOT NULL DEFAULT (datetime('now')),
        last_seen_at DATETIME NOT NULL DEFAULT (datetime('now')),
        request_count INTEGER NOT NULL DEFAULT 1,
        UNIQUE (ip_address, user_agent)
    );
