-- name: AddFavorite :exec
INSERT INTO
    photo_favorites (device_serial, rel_path)
VALUES
    (?, ?)
ON CONFLICT (device_serial, rel_path) DO NOTHING;

-- name: RemoveFavorite :exec
DELETE FROM photo_favorites
WHERE
    device_serial = ?
    AND rel_path = ?;

-- name: IsFavorite :one
SELECT
    COUNT(*) > 0
FROM
    photo_favorites
WHERE
    device_serial = ?
    AND rel_path = ?;

-- name: ListFavorites :many
SELECT
    *
FROM
    photo_favorites
ORDER BY
    created_at DESC;

-- name: CreateFavoritesAlbum :one
INSERT INTO
    photo_albums (name, smart_type)
VALUES
    ('Favorites', 'favorites')
RETURNING *;

-- name: GetFavoritesAlbum :one
SELECT
    *
FROM
    photo_albums
WHERE
    smart_type = 'favorites'
LIMIT
    1;


