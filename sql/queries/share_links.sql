-- name: CreateShareLink :one
INSERT INTO share_links (token, device_serial, rel_path, password_hash, max_uses, expires_at, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: GetShareLinkByToken :one
SELECT * FROM share_links WHERE token = ? LIMIT 1;

-- name: IncrementShareLinkUseCount :exec
UPDATE share_links SET use_count = use_count + 1 WHERE token = ?;

-- name: ListShareLinks :many
SELECT * FROM share_links ORDER BY created_at DESC LIMIT 100;

-- name: DeleteShareLink :exec
DELETE FROM share_links WHERE id = ?;

-- name: DeleteExpiredShareLinks :exec
DELETE FROM share_links
WHERE expires_at IS NOT NULL AND expires_at < datetime('now');
