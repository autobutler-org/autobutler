-- ListJobs lists every job of the given kinds, newest first. It and
-- ListUserJobs are written once per engine, because an IN-list over a
-- parameter has no form both accept (rule 15 in sql/queries/AGENTS.md). This
-- is the SQLite form and sql/queries/postgres/jobs_postgres.sql the PostgreSQL
-- one: change both together.
-- name: ListJobs :many
SELECT
    *
FROM
    jobs
WHERE
    kind IN (sqlc.slice('kinds'))
ORDER BY
    id DESC;

-- ListJobs for one account, so a non-admin's listing never loads every other
-- account's jobs (#2756).
-- user_id stays ahead of the slice: SQLite numbers the slice's bare
-- placeholders from the highest number before them, so a named parameter
-- written after it would bind a kind (sqlc-dev/sqlc#2452).
-- name: ListUserJobs :many
SELECT
    *
FROM
    jobs
WHERE
    user_id = sqlc.arg(user_id)
    AND kind IN (sqlc.slice('kinds'))
ORDER BY
    id DESC;
