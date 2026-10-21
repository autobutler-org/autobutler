-- name: GetInstallID :one
SELECT install_id FROM install WHERE id = 1;

-- For a reset that leaves the database standing (files or devices only): the
-- other instances still have to drop what they hold for the wiped install.
-- The id is made in Go (db.NewInstallID), so no engine needs a random-bytes
-- function.
-- name: RotateInstallID :exec
UPDATE install SET install_id = sqlc.arg(install_id) WHERE id = 1;

-- LockInstall changes nothing: it is a no-op UPDATE run for the row lock it
-- takes on the install's one row, which every database has from its first
-- migration, as the first statement of first-boot setup (#3647). Before setup
-- there is no user row to lock, so two setups at READ COMMITTED on PostgreSQL
-- would each count no users and both create a founding admin; with the lock
-- the second waits here until the first commits and its count sees the first's
-- user. On SQLite the transaction already holds the write lock (BEGIN
-- IMMEDIATE), and this changes nothing.
-- name: LockInstall :exec
UPDATE install SET install_id = install_id WHERE id = 1;
