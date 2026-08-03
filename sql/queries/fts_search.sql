-- name: UpsertFTSDocument :exec
INSERT INTO fts_documents (device_serial, rel_path, content_hash)
VALUES (?, ?, ?)
ON CONFLICT (device_serial, rel_path)
DO UPDATE SET content_hash = excluded.content_hash,
              indexed_at   = datetime('now');

-- name: GetFTSDocument :one
SELECT * FROM fts_documents
WHERE device_serial = ? AND rel_path = ?
LIMIT 1;

-- name: DeleteFTSDocument :exec
DELETE FROM fts_documents WHERE device_serial = ? AND rel_path = ?;

-- name: CountFTSDocuments :one
SELECT COUNT(*) FROM fts_documents;
