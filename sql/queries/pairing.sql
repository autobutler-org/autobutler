-- name: CreatePairingToken :exec
INSERT INTO pairing_tokens (token_hash, created_by)
VALUES (?, ?);

-- name: GetPairingToken :one
SELECT * FROM pairing_tokens
WHERE token_hash = ?
  AND expires_at > datetime('now')
  AND used_at IS NULL
LIMIT 1;

-- name: ConsumePairingToken :exec
UPDATE pairing_tokens
SET used_at = datetime('now')
WHERE token_hash = ?;

-- name: PurgePairingTokens :exec
DELETE FROM pairing_tokens
WHERE expires_at <= datetime('now') OR used_at IS NOT NULL;
