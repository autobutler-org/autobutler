CREATE TABLE IF NOT EXISTS vault_location (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    device_serial TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT OR IGNORE INTO vault_location (id, device_serial) VALUES (1, '');
