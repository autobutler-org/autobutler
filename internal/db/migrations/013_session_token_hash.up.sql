-- Migration 013: hash session tokens at rest (#1334)
--
-- Prior to this migration, session tokens were stored as raw hex strings.
-- From this point forward, only the SHA-256 digest of each token is stored.
-- All existing sessions are invalid (the raw tokens they reference are gone),
-- so we delete them to avoid confusing lookup failures. Users will need to
-- log in again after this upgrade.
DELETE FROM sessions;
