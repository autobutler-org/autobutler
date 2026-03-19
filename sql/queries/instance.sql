-- name: GetInstanceID :one
SELECT instance_id FROM instance WHERE id = 1 LIMIT 1;

-- name: InsertInstanceID :exec
INSERT OR IGNORE INTO instance (id, instance_id) VALUES (1, ?);
