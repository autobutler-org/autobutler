-- Short-lived pairing tokens for QR-code based device onboarding.
--
-- A token encodes the full pairing URL that is displayed as a QR code at
-- /mobile. Scanning with a phone exchanges the token for a session.
-- Tokens are single-use and expire after 10 minutes.
CREATE TABLE IF NOT EXISTS pairing_tokens (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash TEXT NOT NULL UNIQUE,           -- SHA-256 of the raw token
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    expires_at DATETIME NOT NULL DEFAULT (datetime('now', '+10 minutes')),
    used_at    DATETIME                        -- set on first use; token then rejected
);

CREATE INDEX IF NOT EXISTS idx_pairing_tokens_token_hash  ON pairing_tokens (token_hash);
CREATE INDEX IF NOT EXISTS idx_pairing_tokens_expires_at  ON pairing_tokens (expires_at);
