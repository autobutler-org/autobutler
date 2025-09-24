-- name: CreateCalendarEvent :one
INSERT INTO
    calendar_events (
        title,
        description,
        start_time,
        end_time,
        all_day,
        location,
        calendar_id
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetCalendarEvent :one
SELECT
    *
FROM
    calendar_events
WHERE
    id = ?
LIMIT
    1;

-- name: ListCalendarEvents :many
SELECT
    *
FROM
    calendar_events
ORDER BY
    start_time;

-- name: UpdateCalendarEvent :one
UPDATE calendar_events
SET
    title = ?,
    description = ?,
    start_time = ?,
    end_time = ?,
    all_day = ?,
    location = ?,
    calendar_id = ?
WHERE
    id = ? RETURNING *;

-- name: DeleteCalendarEvent :exec
DELETE FROM calendar_events
WHERE
    id = ?;
