-- name: GetDeviceName :one
SELECT display_name FROM device_names WHERE device_path = ? LIMIT 1;

-- name: UpsertDeviceName :exec
INSERT INTO device_names (device_path, display_name, updated_at)
VALUES (?, ?, datetime('now'))
ON CONFLICT(device_path) DO UPDATE SET
    display_name = excluded.display_name,
    updated_at = excluded.updated_at;

-- name: DeleteDeviceName :exec
DELETE FROM device_names WHERE device_path = ?;
