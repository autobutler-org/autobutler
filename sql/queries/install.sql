-- name: GetInstallID :one
SELECT install_id FROM install WHERE id = 1;

-- For a reset that leaves the database standing (files or devices only): the
-- other instances still have to drop what they hold for the wiped install.
-- The id is made in Go (db.NewInstallID), so no engine needs a random-bytes
-- function.
-- name: RotateInstallID :exec
UPDATE install SET install_id = sqlc.arg(install_id) WHERE id = 1;
