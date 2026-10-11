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
