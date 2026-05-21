package db

import "database/sql"

const VaultSchemaDDL = `
CREATE TABLE IF NOT EXISTS vault_config (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    salt BLOB NOT NULL,
    argon2_memory INTEGER NOT NULL DEFAULT 65536,
    argon2_iterations INTEGER NOT NULL DEFAULT 3,
    argon2_parallelism INTEGER NOT NULL DEFAULT 4,
    verification_blob BLOB NOT NULL,
    verification_nonce BLOB NOT NULL,
    auto_lock_seconds INTEGER NOT NULL DEFAULT 900,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS vault_folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    parent_id INTEGER REFERENCES vault_folders(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS vault_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    url_host TEXT NOT NULL DEFAULT '',
    folder_id INTEGER REFERENCES vault_folders(id) ON DELETE SET NULL,
    ciphertext BLOB NOT NULL,
    nonce BLOB NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_vault_entries_folder ON vault_entries(folder_id);
CREATE INDEX IF NOT EXISTS idx_vault_entries_url_host ON vault_entries(url_host);
`

func InitVaultSchema(d *sql.DB) error {
	_, err := d.Exec(VaultSchemaDDL)
	return err
}
