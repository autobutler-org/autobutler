-- name: RegisterDevice :one
INSERT INTO registered_devices (name, device_type, identity_type, ip_address, mac_address, tailscale_key, user_agent)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT DO UPDATE SET
    name         = excluded.name,
    device_type  = excluded.device_type,
    user_agent   = excluded.user_agent,
    updated_at   = datetime('now')
RETURNING *;

-- name: RegisterTailscaleDevice :one
INSERT INTO registered_devices (name, device_type, identity_type, ip_address, tailscale_key, user_agent)
VALUES (?, ?, 'tailscale', ?, ?, ?)
ON CONFLICT DO UPDATE SET
    name        = excluded.name,
    device_type = excluded.device_type,
    ip_address  = excluded.ip_address,
    user_agent  = excluded.user_agent,
    updated_at  = datetime('now')
RETURNING *;

-- name: GetRegisteredDevice :one
SELECT * FROM registered_devices WHERE id = ? LIMIT 1;

-- name: GetRegisteredDeviceByIPUA :one
SELECT * FROM registered_devices
WHERE ip_address = ? AND user_agent = ? AND tailscale_key IS NULL
LIMIT 1;

-- name: GetRegisteredDeviceByTSKey :one
SELECT * FROM registered_devices
WHERE tailscale_key = ?
LIMIT 1;

-- name: ListRegisteredDevices :many
SELECT * FROM registered_devices ORDER BY created_at DESC;

-- name: ListRegisteredDevicesByStatus :many
SELECT * FROM registered_devices WHERE approval_status = ? ORDER BY created_at DESC;

-- name: ApproveDevice :one
UPDATE registered_devices
SET
    approval_status = 'approved',
    approved_by     = ?,
    approved_at     = datetime('now'),
    updated_at      = datetime('now')
WHERE id = ?
RETURNING *;

-- name: RevokeDevice :one
UPDATE registered_devices
SET
    approval_status = 'revoked',
    approved_by     = ?,
    approved_at     = datetime('now'),
    updated_at      = datetime('now')
WHERE id = ?
RETURNING *;

-- name: DeleteRegisteredDevice :exec
DELETE FROM registered_devices WHERE id = ?;

-- name: CountPendingDevices :one
SELECT COUNT(*) FROM registered_devices WHERE approval_status = 'pending';
