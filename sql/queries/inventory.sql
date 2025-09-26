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

-- name: GetInventoryByName :one
SELECT
    *
FROM
    inventory
WHERE
    name = ?
LIMIT
    1;

-- name: ListInventories :many
SELECT
    *
FROM
    inventory
ORDER BY
    name;

-- name: UpdateInventory :one
UPDATE inventory
SET
    name = ?,
    amount = ?,
    unit = ?
WHERE
    id = ? RETURNING *;

-- name: DeleteInventory :exec
DELETE FROM inventory
WHERE
    id = ?;
