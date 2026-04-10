-- name: CreateAlbum :one
INSERT INTO
    photo_albums (name, parent_id)
VALUES
    (?, ?)
RETURNING *;

-- name: GetAlbum :one
SELECT
    *
FROM
    photo_albums
WHERE
    id = ?
LIMIT
    1;

-- name: ListAlbums :many
SELECT
    *
FROM
    photo_albums
ORDER BY
    parent_id,
    name;

-- name: ListRootAlbums :many
SELECT
    *
FROM
    photo_albums
WHERE
    parent_id IS NULL
ORDER BY
    name;

-- name: ListChildAlbums :many
SELECT
    *
FROM
    photo_albums
WHERE
    parent_id = ?
ORDER BY
    name;

-- name: RenameAlbum :one
UPDATE photo_albums
SET
    name = ?,
    updated_at = datetime('now')
WHERE
    id = ?
RETURNING *;

-- name: MoveAlbum :one
UPDATE photo_albums
SET
    parent_id = ?,
    updated_at = datetime('now')
WHERE
    id = ?
RETURNING *;

-- name: DeleteAlbum :exec
DELETE FROM photo_albums
WHERE
    id = ?;

-- name: AddPhotoToAlbum :one
INSERT INTO
    photo_album_items (album_id, device_serial, rel_path)
VALUES
    (?, ?, ?)
ON CONFLICT (album_id, device_serial, rel_path) DO NOTHING
RETURNING *;

-- name: RemovePhotoFromAlbum :exec
DELETE FROM photo_album_items
WHERE
    album_id = ?
    AND device_serial = ?
    AND rel_path = ?;

-- name: ListAlbumItems :many
SELECT
    *
FROM
    photo_album_items
WHERE
    album_id = ?
ORDER BY
    added_at DESC;

-- name: CountAlbumItems :one
SELECT
    COUNT(*)
FROM
    photo_album_items
WHERE
    album_id = ?;

-- name: GetAlbumCoverItem :one
SELECT
    *
FROM
    photo_album_items
WHERE
    album_id = ?
ORDER BY
    added_at DESC
LIMIT
    1;

-- name: ListAlbumsContainingPhoto :many
SELECT
    pa.*
FROM
    photo_albums pa
    JOIN photo_album_items pai ON pa.id = pai.album_id
WHERE
    pai.device_serial = ?
    AND pai.rel_path = ?
ORDER BY
    pa.name;
