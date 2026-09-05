-- One row per client that has talked to this appliance, used by the
-- connected-devices view. Not storage: these are HTTP peers, keyed by the
-- address and user agent they present.
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
