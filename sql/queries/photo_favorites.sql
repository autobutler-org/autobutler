-- name: AddFavorite :exec
INSERT INTO
    photo_favorites (user_id, device_serial, rel_path)
VALUES
    (sqlc.arg(user_id), sqlc.arg(device_serial), sqlc.arg(rel_path))
ON CONFLICT (user_id, device_serial, rel_path) DO NOTHING;

-- name: RemoveFavorite :exec
DELETE FROM photo_favorites
WHERE
    user_id = sqlc.arg(user_id)
    AND device_serial = sqlc.arg(device_serial)
    AND rel_path = sqlc.arg(rel_path);

-- DeleteFavoritesUnder drops the favorite for a deleted path, and every
-- favorite under it when the path is a folder, for every account.
-- name: DeleteFavoritesUnder :exec
DELETE FROM photo_favorites
WHERE
    device_serial = sqlc.arg(device_serial)
    AND (
        rel_path = sqlc.arg(rel_path)
        OR substr(rel_path, 1, length(sqlc.arg(rel_path)) + 1) = sqlc.arg(rel_path) || '/'
    );

-- MoveFavorites points favorites at a moved file or folder, for every account.
-- The count guard skips a row whose destination is already that account's favorite;
-- DeleteFavoritesUnder on the old path clears those leftovers.
-- It replaces UPDATE OR IGNORE, which PostgreSQL has no form of. COUNT(*) = 0
-- and not NOT EXISTS: sqlc v1.30.0's SQLite engine leaves a parameter under a
-- NOT unexpanded (sqlc-dev/sqlc#4110; v1.31.1 expands it).
-- name: MoveFavorites :exec
UPDATE photo_favorites
SET
    device_serial = sqlc.arg(new_device_serial),
    rel_path = sqlc.arg(new_rel_path) || substr(photo_favorites.rel_path, length(CAST(sqlc.arg(old_rel_path) AS TEXT)) + 1)
WHERE
    photo_favorites.device_serial = sqlc.arg(old_device_serial)
    AND (
        photo_favorites.rel_path = sqlc.arg(old_rel_path)
        OR substr(photo_favorites.rel_path, 1, length(sqlc.arg(old_rel_path)) + 1) = sqlc.arg(old_rel_path) || '/'
    )
    AND (
        SELECT
            COUNT(*)
        FROM
            photo_favorites AS held
        WHERE
            held.user_id = photo_favorites.user_id
            AND held.device_serial = sqlc.arg(new_device_serial)
            AND held.rel_path = sqlc.arg(new_rel_path) || substr(photo_favorites.rel_path, length(CAST(sqlc.arg(old_rel_path) AS TEXT)) + 1)
    ) = 0;

-- name: IsFavorite :one
SELECT
    COUNT(*) > 0
FROM
    photo_favorites
WHERE
    user_id = sqlc.arg(user_id)
    AND device_serial = sqlc.arg(device_serial)
    AND rel_path = sqlc.arg(rel_path);

-- name: ListFavorites :many
SELECT
    *
FROM
    photo_favorites
WHERE
    user_id = sqlc.arg(user_id)
ORDER BY
    created_at DESC;

-- photo_albums.user_id is nullable only because SQLite cannot add a NOT NULL
-- foreign key column (014); the CAST keeps the parameter a plain id.
-- name: CreateFavoritesAlbum :one
INSERT INTO
    photo_albums (name, smart_type, user_id)
VALUES
    ('Favorites', 'favorites', CAST(sqlc.arg(user_id) AS BIGINT))
RETURNING *;

-- name: GetFavoritesAlbum :one
SELECT
    *
FROM
    photo_albums
WHERE
    user_id = CAST(sqlc.arg(user_id) AS BIGINT)
    AND smart_type = 'favorites'
LIMIT
    1;
