-- name: GetDeviceName :one
SELECT display_name FROM device_names WHERE device_serial = ? LIMIT 1;

-- name: GetAllDeviceNames :many
SELECT device_serial, display_name FROM device_names;

-- name: UpsertDeviceName :exec
INSERT INTO device_names (device_serial, display_name, updated_at)
VALUES (?, ?, datetime('now'))
ON CONFLICT(device_serial) DO UPDATE SET
    display_name = excluded.display_name,
    updated_at = datetime('now');

-- name: DeleteDeviceName :exec
DELETE FROM device_names WHERE device_serial = ?;
