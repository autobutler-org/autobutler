-- name: GetVaultLocation :one
SELECT device_serial FROM vault_location WHERE id = 1;

-- name: SetVaultLocation :exec
UPDATE vault_location SET device_serial = sqlc.arg(device_serial), updated_at = CURRENT_TIMESTAMP WHERE id = 1;
