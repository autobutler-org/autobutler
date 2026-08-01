-- name: CreateShareLink :one
INSERT INTO share_links (
    token_hash, created_by, resource_type, resource_path, device_serial, expires_at
) VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetShareLinkByTokenHash :one
SELECT * FROM share_links WHERE token_hash = ?;

-- name: ListShareLinksByUser :many
SELECT * FROM share_links
WHERE created_by = ?
ORDER BY created_at DESC;

-- name: DeleteShareLink :exec
DELETE FROM share_links WHERE token_hash = ? AND created_by = ?;

-- name: IncrementShareLinkViewCount :exec
UPDATE share_links SET view_count = view_count + 1 WHERE token_hash = ?;

-- name: DeleteExpiredShareLinks :exec
DELETE FROM share_links
WHERE expires_at IS NOT NULL AND expires_at < datetime('now');
