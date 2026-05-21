-- name: GetDeviceRole :one
SELECT role FROM device_roles WHERE device_serial = ? LIMIT 1;

-- name: GetAllDeviceRoles :many
SELECT device_serial, role FROM device_roles;

-- name: UpsertDeviceRole :exec
INSERT INTO device_roles (device_serial, role, updated_at)
VALUES (?, ?, datetime('now'))
ON CONFLICT(device_serial) DO UPDATE SET
    role = excluded.role,
    updated_at = datetime('now');

-- name: ClearDefaultStorageRole :exec
UPDATE device_roles SET role = 'unassigned', updated_at = datetime('now')
WHERE role = 'default-storage';

-- name: DeleteDeviceRole :exec
DELETE FROM device_roles WHERE device_serial = ?;
