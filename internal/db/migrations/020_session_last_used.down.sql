-- Rollback 020: drop the use tracking added for sliding expiry (#1647).
--
-- Sessions survive the rollback; they simply stop renewing and revert to the
-- fixed window their current expires_at already describes.
ALTER TABLE sessions
DROP COLUMN last_used_at;
