-- Migration 020: track session use so expiry can slide (#1647)
--
-- expires_at was stamped once at login and never written again, so a session
-- was a fixed 30-day window from that moment: a user who opened the app every
-- day was still forced back to the login screen on day 31.
--
-- last_used_at is what makes renewal-on-use possible, and what debounces it --
-- ValidateSession only rewrites expires_at once the column is stale enough,
-- rather than on every authenticated request.
--
-- SQLite rejects a non-constant DEFAULT in ALTER TABLE ADD COLUMN, so
-- datetime('now') is not available here. The epoch literal is a placeholder
-- that no row keeps: existing rows are backfilled below, and every new row
-- gets an explicit value from CreateSession. A row that somehow kept it would
-- simply look stale and renew on first use, which is harmless.
ALTER TABLE sessions
ADD COLUMN last_used_at DATETIME NOT NULL DEFAULT '1970-01-01 00:00:00';

-- Sessions predating this migration have never been renewed, so their creation
-- is the last thing we know about them.
UPDATE sessions SET last_used_at = created_at;
