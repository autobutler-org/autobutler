-- name: CreateUser :one
INSERT INTO users (username, password_hash, recovery_phrase_hash)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = ? LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: GetFirstUser :one
SELECT * FROM users ORDER BY id ASC LIMIT 1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = ?
WHERE id = ?;

-- name: CreateSession :one
INSERT INTO sessions (token, token_hash, user_id, expires_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetSession :one
SELECT s.*, u.username
FROM sessions s
JOIN users u ON s.user_id = u.id
WHERE s.token_hash = ? AND s.expires_at > datetime('now')
LIMIT 1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= datetime('now');

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = ?;

-- name: ListActiveSessionsForUser :many
SELECT token, user_id, expires_at, created_at
FROM sessions
WHERE user_id = ? AND expires_at > datetime('now')
ORDER BY created_at DESC;
