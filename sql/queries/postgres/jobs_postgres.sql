-- ListJobs lists every job of the given kinds, newest first. It and
-- ListUserJobs are written once per engine, because an IN-list over a
-- parameter has no form both accept (rule 15 in sql/queries/AGENTS.md). This
-- is the PostgreSQL form and sql/queries/sqlite/jobs_sqlite.sql the SQLite
-- one: change both together.
-- name: ListJobs :many
SELECT
    *
FROM
    jobs
WHERE
    kind = ANY (sqlc.arg(kinds)::TEXT[])
ORDER BY
    id DESC;

-- ListJobs for one account, so a non-admin's listing never loads every other
-- account's jobs (#2756).
-- name: ListUserJobs :many
SELECT
    *
FROM
    jobs
WHERE
    user_id = sqlc.arg(user_id)
    AND kind = ANY (sqlc.arg(kinds)::TEXT[])
ORDER BY
    id DESC;
