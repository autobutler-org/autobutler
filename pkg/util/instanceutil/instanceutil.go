// Package instanceutil manages the butler's unique instance identifier.
//
// The instance ID is a random UUIDv4 generated once on first boot and stored
// in the database. It is exposed via the /auth/status endpoint so Flutter
// clients can detect when they've connected to a different butler than
// expected — e.g. a neighbor's butler at the same LAN hostname.
package instanceutil

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/autobutler-org/autobutler/internal/db"
)

// EnsureInstanceID ensures the instance ID row exists in the database,
// generating a new random ID if one hasn't been created yet.
// This should be called once during server startup after migrations.
func EnsureInstanceID(ctx context.Context, queries *db.Queries) error {
	id, err := GenerateID()
	if err != nil {
		return fmt.Errorf("failed to generate instance ID: %w", err)
	}
	// INSERT OR IGNORE — no-op if the row already exists.
	return queries.InsertInstanceID(ctx, id)
}

// GetInstanceID retrieves the stored instance ID from the database.
func GetInstanceID(ctx context.Context, queries *db.Queries) (string, error) {
	id, err := queries.GetInstanceID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get instance ID: %w", err)
	}
	return id, nil
}

// GenerateID returns a cryptographically random UUID v4 string.
func GenerateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Set version (4) and variant bits per RFC 4122.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%12x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16],
	), nil
}
