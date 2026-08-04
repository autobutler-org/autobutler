-- name: GetUserRole :one
SELECT role FROM users WHERE id = ? LIMIT 1;

-- name: SetUserRole :exec
UPDATE users SET role = ? WHERE id = ?;

-- name: IsAdmin :one
-- Returns 1 if the user has owner or admin role, 0 otherwise.
SELECT CASE WHEN role IN ('owner', 'admin') THEN 1 ELSE 0 END FROM users WHERE id = ? LIMIT 1;

-- name: CountAdmins :one
-- Counts users with owner or admin role. Used to prevent demoting the last admin.
SELECT COUNT(*) FROM users WHERE role IN ('owner', 'admin');

-- name: ListUserRoles :many
SELECT id, username, role, created_at FROM users ORDER BY id ASC;
