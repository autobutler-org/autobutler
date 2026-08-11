package favoritesutil

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
)

// isUniqueConstraintErr is unexported, so it can only be exercised from inside
// the package. The external favoritesutil_test package covers the exported
// surface; this file covers the helper that decides whether a failed insert was
// a benign race (two callers creating the Favorites album at once) or a real
// error worth propagating.
func TestIsUniqueConstraintErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil is not a constraint error", nil, false},
		{"ErrNoRows is not a constraint error", sql.ErrNoRows, false},
		{"unrelated error", errors.New("connection refused"), false},
		{"bare UNIQUE constraint failed", errors.New("UNIQUE constraint failed"), true},
		{"driver-prefixed constraint error",
			errors.New("constraint failed: UNIQUE constraint failed: photo_albums.name"), true},
		{"wrapped constraint error",
			fmt.Errorf("ensure album: %w", errors.New("UNIQUE constraint failed: photo_albums.name")), true},
		{"lowercase does not match", errors.New("unique constraint failed"), false},
		{"other constraint kinds do not match",
			errors.New("NOT NULL constraint failed: photo_albums.name"), false},
		{"FOREIGN KEY constraint does not match",
			errors.New("FOREIGN KEY constraint failed"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUniqueConstraintErr(tt.err); got != tt.want {
				t.Errorf("isUniqueConstraintErr(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
