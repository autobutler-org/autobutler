-- The per-account, end-to-end encrypted vault (#3456, #3458). The client
-- encrypts everything; the Quark stores ciphertext and nothing it could
-- read. The household vault's tables (005_vault) are dropped once the last
-- code reading them goes (#3460).

-- Each account's vault key, a random 32-byte key made on the client, stored
-- only wrapped: under a key derived from the login password and, when the
-- client had it, from the recovery phrase, exactly as user_chat_keys stores
-- the chat identity. kdf_params is the client's JSON record of the Argon2id
-- cost, opaque here.
CREATE TABLE vault_keys (
    user_id             INTEGER PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    wrapped_by_password BLOB NOT NULL,
    salt_pw             BLOB NOT NULL,
    -- Null for a key wrapped at a sign-in that had no recovery phrase to hand.
    wrapped_by_phrase   BLOB,
    salt_rp             BLOB,
    kdf_params          TEXT NOT NULL,
    created_at          DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at          DATETIME NOT NULL DEFAULT (datetime('now')),
    CHECK ((wrapped_by_phrase IS NULL) = (salt_rp IS NULL))
);

-- Each account's vault items, entries and folders alike, encrypted under the
-- vault key. What kind of item a row is, its name, URL, username, password,
-- notes, TOTP secret and folder all live inside ciphertext, so no column
-- says anything about the content. format_version is the client's version
-- of the plaintext layout.
CREATE TABLE vault_items (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id        INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    ciphertext     BLOB NOT NULL,
    nonce          BLOB NOT NULL,
    format_version INTEGER NOT NULL,
    created_at     DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at     DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_vault_items_user ON vault_items (user_id);
