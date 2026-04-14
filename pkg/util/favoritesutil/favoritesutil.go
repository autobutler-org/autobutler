package favoritesutil

import (
	"context"
	"database/sql"
	"errors"

	"github.com/autobutler-org/autobutler/internal/db"
)

// EnsureFavoritesAlbum returns the system Favorites album, creating it if it
// doesn't exist. This should be called on server startup so the album is always
// present in the sidebar even when empty.
func EnsureFavoritesAlbum(ctx context.Context, q *db.Queries) (db.PhotoAlbum, error) {
	album, err := q.GetFavoritesAlbum(ctx)
	if err == nil {
		return album, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return db.PhotoAlbum{}, err
	}
	// Create it
	return q.CreateFavoritesAlbum(ctx)
}

// ToggleFavorite adds or removes a photo from favorites and syncs the
// Favorites smart album. Returns true if the photo is now favorited.
func ToggleFavorite(ctx context.Context, q *db.Queries, deviceSerial, relPath string) (bool, error) {
	isFav, err := q.IsFavorite(ctx, db.IsFavoriteParams{
		DeviceSerial: deviceSerial,
		RelPath:      relPath,
	})
	if err != nil {
		return false, err
	}

	if isFav {
		if err := q.RemoveFavorite(ctx, db.RemoveFavoriteParams{
			DeviceSerial: deviceSerial,
			RelPath:      relPath,
		}); err != nil {
			return false, err
		}
		// Remove from Favorites album
		album, err := EnsureFavoritesAlbum(ctx, q)
		if err == nil {
			_ = q.RemovePhotoFromAlbum(ctx, db.RemovePhotoFromAlbumParams{
				AlbumID:      album.ID,
				DeviceSerial: deviceSerial,
				RelPath:      relPath,
			})
		}
		return false, nil
	}

	if err := q.AddFavorite(ctx, db.AddFavoriteParams{
		DeviceSerial: deviceSerial,
		RelPath:      relPath,
	}); err != nil {
		return false, err
	}
	// Add to Favorites album
	album, err := EnsureFavoritesAlbum(ctx, q)
	if err == nil {
		_, _ = q.AddPhotoToAlbum(ctx, db.AddPhotoToAlbumParams{
			AlbumID:      album.ID,
			DeviceSerial: deviceSerial,
			RelPath:      relPath,
		})
	}
	return true, nil
}
