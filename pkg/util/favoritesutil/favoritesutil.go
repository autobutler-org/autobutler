package favoritesutil

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"

	"github.com/autobutler-org/quark/internal/db"
)

// isUniqueConstraintErr reports whether err is a SQLite unique-constraint
// violation (modernc.org/sqlite surfaces these as error strings).
func isUniqueConstraintErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// EnsureFavoritesAlbum returns the system Favorites album, creating it if it
// doesn't exist. Safe to call concurrently — if two goroutines race to create
// the album, the loser re-fetches the winner's row.
func EnsureFavoritesAlbum(ctx context.Context, q *db.Queries) (db.PhotoAlbum, error) {
	album, err := q.GetFavoritesAlbum(ctx)
	if err == nil {
		return album, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return db.PhotoAlbum{}, err
	}
	// Attempt to create — handle the race where another request creates it
	// concurrently and hits the unique constraint on smart_type.
	album, err = q.CreateFavoritesAlbum(ctx)
	if err == nil {
		return album, nil
	}
	if isUniqueConstraintErr(err) {
		// Another request won the race; re-fetch the album they created.
		return q.GetFavoritesAlbum(ctx)
	}
	return db.PhotoAlbum{}, err
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
		// Best-effort: remove from the Favorites smart album. A failure here
		// leaves a stale album row but does not affect the authoritative
		// photo_favorites table, so we log and continue rather than rolling back.
		album, err := EnsureFavoritesAlbum(ctx, q)
		if err == nil {
			if removeErr := q.RemovePhotoFromAlbum(ctx, db.RemovePhotoFromAlbumParams{
				AlbumID:      album.ID,
				DeviceSerial: deviceSerial,
				RelPath:      relPath,
			}); removeErr != nil && !errors.Is(removeErr, sql.ErrNoRows) {
				log.Printf("[favorites] best-effort album remove failed for %q: %v", relPath, removeErr)
			}
		}
		return false, nil
	}

	if err := q.AddFavorite(ctx, db.AddFavoriteParams{
		DeviceSerial: deviceSerial,
		RelPath:      relPath,
	}); err != nil {
		return false, err
	}
	// Best-effort: add to the Favorites smart album. A duplicate-key error
	// means it's already there (idempotent); any other failure is logged but
	// does not fail the toggle since photo_favorites is the source of truth.
	album, err := EnsureFavoritesAlbum(ctx, q)
	if err == nil {
		if _, addErr := q.AddPhotoToAlbum(ctx, db.AddPhotoToAlbumParams{
			AlbumID:      album.ID,
			DeviceSerial: deviceSerial,
			RelPath:      relPath,
		}); addErr != nil && !isUniqueConstraintErr(addErr) {
			log.Printf("[favorites] best-effort album add failed for %q: %v", relPath, addErr)
		}
	}
	return true, nil
}
