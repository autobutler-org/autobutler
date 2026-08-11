-- name: SetTOTPPending :exec
UPDATE users SET totp_pending = ? WHERE id = ?;

-- name: ConfirmTOTP :exec
UPDATE users SET totp_secret = totp_pending, totp_pending = NULL WHERE id = ?;

-- name: DisableTOTP :exec
UPDATE users SET totp_secret = NULL, totp_pending = NULL WHERE id = ?;

-- name: GetTOTPSecret :one
SELECT totp_secret FROM users WHERE id = ?;

-- name: GetTOTPPending :one
SELECT totp_pending FROM users WHERE id = ?;

-- name: CreateTOTPChallenge :exec
INSERT INTO totp_challenges (user_id, token_hash)
VALUES (?, ?);

-- name: GetTOTPChallenge :one
SELECT * FROM totp_challenges
WHERE token_hash = ? AND expires_at > datetime('now')
LIMIT 1;

-- name: DeleteTOTPChallenge :exec
DELETE FROM totp_challenges WHERE token_hash = ?;

-- name: PurgeTOTPChallenges :exec
DELETE FROM totp_challenges WHERE expires_at <= datetime('now');
