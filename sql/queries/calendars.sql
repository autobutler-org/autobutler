-- name: CreateCalendar :one
INSERT INTO
    calendars (name)
VALUES
    (?) RETURNING *;

-- name: GetCalendar :one
SELECT
    *
FROM
    calendars
WHERE
    id = ?
LIMIT
    1;

-- name: ListCalendars :many
SELECT
    *
FROM
    calendars
ORDER BY
    name;

-- name: UpdateCalendar :one
UPDATE calendars
SET
    name = ?
WHERE
    id = ? RETURNING *;

-- name: DeleteCalendar :exec
DELETE FROM calendars
WHERE
    id = ?;
