-- name: GetVaultLocation :one
SELECT device_serial FROM vault_location WHERE id = 1;

-- name: SetVaultLocation :exec
UPDATE vault_location SET device_serial = ?, updated_at = datetime('now') WHERE id = 1;
