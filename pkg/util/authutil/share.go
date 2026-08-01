package authutil

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
)

// ShareLinkParams describes a new share link to create.
type ShareLinkParams struct {
	UserID       int64
	ResourceType string // "file" or "folder"
	ResourcePath string
	DeviceSerial string
	ExpiresAt    *time.Time // nil = never expires
}

// CreateShareLink generates a random token, stores its hash, and returns the raw token.
func CreateShareLink(ctx context.Context, queries *db.Queries, p ShareLinkParams) (string, error) {
	raw, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generate share token: %w", err)
	}
	hash := hashToken(raw)

	expires := sql.NullTime{}
	if p.ExpiresAt != nil {
		expires = sql.NullTime{Time: *p.ExpiresAt, Valid: true}
	}

	if _, err := queries.CreateShareLink(ctx, db.CreateShareLinkParams{
		TokenHash:    hash,
		CreatedBy:    sql.NullInt64{Int64: p.UserID, Valid: true},
		ResourceType: p.ResourceType,
		ResourcePath: p.ResourcePath,
		DeviceSerial: p.DeviceSerial,
		ExpiresAt:    expires,
	}); err != nil {
		return "", fmt.Errorf("store share link: %w", err)
	}
	return raw, nil
}

// ValidateShareLink looks up and validates a raw token. Returns the share link on success.
// Returns an error if the token is invalid or has expired.
func ValidateShareLink(ctx context.Context, queries *db.Queries, rawToken string) (*db.ShareLink, error) {
	hash := hashToken(rawToken)
	link, err := queries.GetShareLinkByTokenHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("share link not found")
	}
	if link.ExpiresAt.Valid && time.Now().After(link.ExpiresAt.Time) {
		return nil, fmt.Errorf("share link has expired")
	}
	return &link, nil
}

// RevokeShareLink deletes a share link owned by userID.
func RevokeShareLink(ctx context.Context, queries *db.Queries, rawToken string, userID int64) error {
	hash := hashToken(rawToken)
	return queries.DeleteShareLink(ctx, db.DeleteShareLinkParams{
		TokenHash: hash,
		CreatedBy: sql.NullInt64{Int64: userID, Valid: true},
	})
}

// generateToken returns a 192-bit cryptographically random URL-safe token.
func generateToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken lives in service.go — shared by session, challenge, and share tokens.
