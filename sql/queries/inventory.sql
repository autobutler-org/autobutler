-- name: CreateInventory :one
INSERT INTO
    inventory (name, amount, unit)
VALUES
    (?, ?, ?) RETURNING *;

-- name: GetInventory :one
SELECT
    *
FROM
    inventory
WHERE
    id = ?
LIMIT
    1;

-- name: ListInventories :many
SELECT
    *
FROM
    inventory
ORDER BY
    name;

-- name: UpdateInventory :exec
UPDATE inventory
SET
    name = ?,
    amount = ?,
    unit = ?
WHERE
    id = ?;

-- name: DeleteInventory :exec
DELETE FROM inventory
WHERE
    id = ?;