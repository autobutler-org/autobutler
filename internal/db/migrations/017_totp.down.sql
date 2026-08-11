DROP INDEX IF EXISTS idx_totp_challenges_expires_at;
DROP INDEX IF EXISTS idx_totp_challenges_token_hash;
DROP TABLE IF EXISTS totp_challenges;
-- SQLite does not support DROP COLUMN on older versions; these are left as
-- no-ops in the down migration. The columns are nullable and harmless.
