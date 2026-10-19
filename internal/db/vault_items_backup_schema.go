package db

// VaultItemsBackupVersion is the format of the vault backup file, stored in
// its user_version. It moves when VaultItemsBackupSchemaDDL does, and a
// restore refuses a version it does not know rather than guess at its columns.
const VaultItemsBackupVersion = 1

// VaultItemsBackupSchemaDDL is the schema of the vault backup file pkg/backup
// writes beside a snapshot (#3460), vault_items_backup.db. Like
// ChatBackupSchemaDDL it is a transport format: each vault table has the live
// table's columns in the live table's order and none of its constraints, which
// the live schema enforces again when a restore loads the rows.
//
// Everything in it is ciphertext or the wrapped keys that open it, so it needs
// no password of its own: an account's vault key opens with that account's
// password or recovery phrase, exactly as it does on the live Quark. users is
// the directory a restore matches accounts through, and holds no hash.
//
// A migration that changes vault_keys or vault_items changes this too, and
// bumps VaultItemsBackupVersion; TestVaultBackupSchema_MatchesLiveTables in
// pkg/backup fails until the columns agree.
const VaultItemsBackupSchemaDDL = `
PRAGMA user_version = 1;

CREATE TABLE users (
    id       INTEGER PRIMARY KEY,
    username TEXT NOT NULL
);

CREATE TABLE vault_keys (
    user_id             INTEGER PRIMARY KEY,
    wrapped_by_password BLOB NOT NULL,
    salt_pw             BLOB NOT NULL,
    wrapped_by_phrase   BLOB,
    salt_rp             BLOB,
    kdf_params          TEXT NOT NULL,
    created_at          DATETIME NOT NULL,
    updated_at          DATETIME NOT NULL
);

CREATE TABLE vault_items (
    id             INTEGER PRIMARY KEY,
    user_id        INTEGER NOT NULL,
    ciphertext     BLOB NOT NULL,
    nonce          BLOB NOT NULL,
    format_version INTEGER NOT NULL,
    created_at     DATETIME NOT NULL,
    updated_at     DATETIME NOT NULL
);
`
