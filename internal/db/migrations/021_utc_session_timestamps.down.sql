-- Rollback 021: nothing to undo (#1650).
--
-- The migration removed rows rather than changing the schema, and the sessions
-- it removed cannot be recreated. Rolling back reverts the storage format for
-- future writes only, which is a code-level concern (the DSN), not a schema
-- one. Sessions written while the fix was in place stay readable either way --
-- the driver parses SQLite's canonical format as happily as Go's.
SELECT 1;
