-- name: SetUserAdmin :exec
UPDATE users SET is_admin = ? WHERE username = ?;

-- name: GetAdminCount :one
SELECT COUNT(*) FROM users WHERE is_admin = 1;

-- name: IsUserAdmin :one
SELECT is_admin FROM users WHERE username = ?;

-- name: ListUsers :many
SELECT id, username, is_admin, created_at FROM users ORDER BY created_at ASC;

-- name: PromoteToAdmin :one
UPDATE users SET is_admin = 1 WHERE username = ? RETURNING id, username, is_admin, created_at;

-- name: DemoteFromAdmin :one
UPDATE users SET is_admin = 0 WHERE username = ? RETURNING id, username, is_admin, created_at;
