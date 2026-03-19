-- Stores the unique instance identifier for this butler.
-- Generated once on first boot; used by clients to detect when they've
-- connected to a different butler than expected (e.g. neighbor's butler
-- on the same LAN hostname).
CREATE TABLE IF NOT EXISTS instance (
    id INTEGER PRIMARY KEY CHECK (id = 1), -- singleton row
    instance_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);
