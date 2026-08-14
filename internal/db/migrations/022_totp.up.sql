-- TOTP 2FA support.
--
-- totp_secret: the base32 TOTP secret (null = 2FA not enrolled).
-- totp_pending: a pending (unconfirmed) secret during enrollment flow.
--
-- Two-phase enrollment: generate → store in totp_pending → user confirms
-- with a valid code → copy to totp_secret, clear totp_pending.
-- This prevents locking a user out if they abandon the enrollment flow.
ALTER TABLE users ADD COLUMN totp_secret  TEXT;
ALTER TABLE users ADD COLUMN totp_pending TEXT;

-- Short-lived challenge tokens issued after password verification when 2FA is
-- required, consumed by the /auth/totp/verify endpoint.
CREATE TABLE IF NOT EXISTS totp_challenges (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL DEFAULT (datetime('now', '+5 minutes'))
);

CREATE INDEX IF NOT EXISTS idx_totp_challenges_token_hash  ON totp_challenges (token_hash);
CREATE INDEX IF NOT EXISTS idx_totp_challenges_expires_at  ON totp_challenges (expires_at);
