-- name: CreateVaultConfig :exec
INSERT INTO vault_config (
    id, salt, argon2_memory, argon2_iterations, argon2_parallelism,
    verification_blob, verification_nonce, auto_lock_seconds
) VALUES (1, sqlc.arg(salt), sqlc.arg(argon2_memory), sqlc.arg(argon2_iterations), sqlc.arg(argon2_parallelism), sqlc.arg(verification_blob), sqlc.arg(verification_nonce), sqlc.arg(auto_lock_seconds));

-- name: GetVaultConfig :one
SELECT * FROM vault_config WHERE id = 1;

-- name: CreateVaultFolder :one
INSERT INTO vault_folders (name, parent_id, sort_order)
VALUES (sqlc.arg(name), sqlc.arg(parent_id), sqlc.arg(sort_order))
RETURNING *;

-- name: ListVaultFolders :many
SELECT * FROM vault_folders ORDER BY sort_order, name;

-- name: GetVaultFolder :one
SELECT * FROM vault_folders WHERE id = sqlc.arg(id);

-- name: UpdateVaultFolder :exec
UPDATE vault_folders
SET name = sqlc.arg(name), parent_id = sqlc.arg(parent_id), sort_order = sqlc.arg(sort_order)
WHERE id = sqlc.arg(id);

-- name: DeleteVaultFolder :exec
DELETE FROM vault_folders WHERE id = sqlc.arg(id);

-- name: CreateVaultEntry :one
INSERT INTO vault_entries (name, url_host, folder_id, ciphertext, nonce)
VALUES (sqlc.arg(name), sqlc.arg(url_host), sqlc.arg(folder_id), sqlc.arg(ciphertext), sqlc.arg(nonce))
RETURNING *;

-- name: GetVaultEntry :one
SELECT * FROM vault_entries WHERE id = sqlc.arg(id);

-- name: ListVaultEntries :many
SELECT id, name, url_host, folder_id, created_at, updated_at
FROM vault_entries
ORDER BY name;

-- name: UpdateVaultEntry :exec
UPDATE vault_entries
SET name = sqlc.arg(name), url_host = sqlc.arg(url_host), folder_id = sqlc.arg(folder_id), ciphertext = sqlc.arg(ciphertext), nonce = sqlc.arg(nonce),
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id);

-- name: DeleteVaultEntry :exec
DELETE FROM vault_entries WHERE id = sqlc.arg(id);

-- name: ListVaultEntriesWithoutHost :many
SELECT id, ciphertext, nonce FROM vault_entries WHERE url_host = '';

-- name: UpdateVaultEntryHost :exec
UPDATE vault_entries SET url_host = sqlc.arg(url_host) WHERE id = sqlc.arg(id);

-- name: ListAllVaultEntriesForReEncrypt :many
SELECT id, ciphertext, nonce FROM vault_entries;

-- name: UpdateVaultEntryCiphertext :exec
UPDATE vault_entries
SET ciphertext = sqlc.arg(ciphertext), nonce = sqlc.arg(nonce), updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id);

-- name: UpdateVaultConfigPassword :exec
UPDATE vault_config
SET salt = sqlc.arg(salt), verification_blob = sqlc.arg(verification_blob), verification_nonce = sqlc.arg(verification_nonce)
WHERE id = 1;

-- The per-account vault (#3458). Every query below takes the account's
-- user_id, so no call can reach another account's rows. The household vault
-- queries above go with their tables (#3460).

-- GetVaultKey returns an account's wrapped vault key. Only the account itself
-- may be served it.
-- name: GetVaultKey :one
SELECT
    *
FROM
    vault_keys
WHERE
    user_id = sqlc.arg(user_id);

-- CreateVaultKey stores an account's first vault key. A second create is a
-- unique violation rather than a replace, so a client can never overwrite the
-- key its items are encrypted under by setting the vault up again.
-- name: CreateVaultKey :one
INSERT INTO vault_keys (
    user_id,
    wrapped_by_password,
    salt_pw,
    wrapped_by_phrase,
    salt_rp,
    kdf_params
) VALUES (
    sqlc.arg(user_id), sqlc.arg(wrapped_by_password), sqlc.arg(salt_pw), sqlc.arg(wrapped_by_phrase), sqlc.arg(salt_rp), sqlc.arg(kdf_params)
)
RETURNING
    *;

-- UpdateVaultKey replaces the wraps of an account's existing vault key, for a
-- password change, a recovery or a new recovery phrase. The key inside them
-- is the same, so the items stay as they are. No row updated means the
-- account has no vault.
-- name: UpdateVaultKey :execrows
UPDATE vault_keys
SET
    wrapped_by_password = sqlc.arg(wrapped_by_password),
    salt_pw = sqlc.arg(salt_pw),
    wrapped_by_phrase = sqlc.arg(wrapped_by_phrase),
    salt_rp = sqlc.arg(salt_rp),
    kdf_params = sqlc.arg(kdf_params),
    updated_at = CURRENT_TIMESTAMP
WHERE
    user_id = sqlc.arg(user_id);

-- DeleteVaultKey removes an account's vault key, which leaves its items
-- unreadable; DeleteVaultItems goes with it.
-- name: DeleteVaultKey :exec
DELETE FROM vault_keys
WHERE
    user_id = sqlc.arg(user_id);

-- ListVaultItems returns every item of an account, oldest first.
-- name: ListVaultItems :many
SELECT
    *
FROM
    vault_items
WHERE
    user_id = sqlc.arg(user_id)
ORDER BY
    id;

-- GetVaultItem returns one of an account's items. Another account's id finds
-- nothing.
-- name: GetVaultItem :one
SELECT
    *
FROM
    vault_items
WHERE
    id = sqlc.arg(id)
    AND user_id = sqlc.arg(user_id);

-- name: CreateVaultItem :one
INSERT INTO vault_items (
    user_id,
    ciphertext,
    nonce,
    format_version
) VALUES (
    sqlc.arg(user_id), sqlc.arg(ciphertext), sqlc.arg(nonce), sqlc.arg(format_version)
)
RETURNING
    *;

-- UpdateVaultItem replaces one of an account's items. Another account's id
-- updates nothing and returns no row.
-- name: UpdateVaultItem :one
UPDATE vault_items
SET
    ciphertext = sqlc.arg(ciphertext),
    nonce = sqlc.arg(nonce),
    format_version = sqlc.arg(format_version),
    updated_at = CURRENT_TIMESTAMP
WHERE
    id = sqlc.arg(id)
    AND user_id = sqlc.arg(user_id)
RETURNING
    *;

-- DeleteVaultItem removes one of an account's items. Another account's id
-- deletes nothing.
-- name: DeleteVaultItem :execrows
DELETE FROM vault_items
WHERE
    id = sqlc.arg(id)
    AND user_id = sqlc.arg(user_id);

-- DeleteVaultItems removes every item of an account.
-- name: DeleteVaultItems :exec
DELETE FROM vault_items
WHERE
    user_id = sqlc.arg(user_id);
