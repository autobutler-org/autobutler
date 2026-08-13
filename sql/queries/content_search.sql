-- name: UpsertFileContent :exec
INSERT INTO file_content (serial, rel_path, extracted, updated_at)
VALUES (?, ?, ?, datetime('now'))
ON CONFLICT(serial, rel_path) DO UPDATE SET
    extracted  = excluded.extracted,
    updated_at = excluded.updated_at;

-- name: DeleteFileContent :exec
DELETE FROM file_content WHERE serial = ? AND rel_path = ?;

-- name: DeleteFileContentBySerial :exec
DELETE FROM file_content WHERE serial = ?;

-- name: GetFileContent :one
SELECT id, serial, rel_path, extracted, updated_at
FROM file_content
WHERE serial = ? AND rel_path = ?;
