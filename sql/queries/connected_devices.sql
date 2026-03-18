-- name: UpsertConnectedDevice :one
INSERT INTO
    connected_devices (ip_address, user_agent, first_seen_at, last_seen_at, request_count)
VALUES
    (?, ?, datetime('now'), datetime('now'), 1)
ON CONFLICT (ip_address, user_agent) DO UPDATE SET
    last_seen_at = datetime('now'),
    request_count = request_count + 1
RETURNING *;

-- name: ListConnectedDevices :many
SELECT
    *
FROM
    connected_devices
ORDER BY
    last_seen_at DESC;

-- name: GetConnectedDevice :one
SELECT
    *
FROM
    connected_devices
WHERE
    id = ?
LIMIT
    1;

-- name: DeleteConnectedDevice :exec
DELETE FROM connected_devices
WHERE
    id = ?;

-- name: CountConnectedDevices :one
SELECT
    COUNT(*)
FROM
    connected_devices;
