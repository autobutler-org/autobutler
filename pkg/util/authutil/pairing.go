package authutil

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/autobutler-org/autobutler/internal/db"
)

// PairingTokenSize is the number of random bytes used for a pairing token.
// 32 bytes → 64 hex chars → ~192 bits of entropy; single-use + 10-min TTL.
const PairingTokenSize = 32

// CreatePairingToken generates a new short-lived pairing token for userID and
// stores its SHA-256 hash in the database. Returns the raw (unhashed) token
// that should be embedded in the QR code URL.
func CreatePairingToken(ctx context.Context, queries *db.Queries, userID int64) (string, error) {
	// Purge stale tokens to keep the table small.
	_ = queries.PurgePairingTokens(ctx)

	raw, err := GenerateSessionToken() // reuse the same CSPRNG helper
	if err != nil {
		return "", fmt.Errorf("generate pairing token: %w", err)
	}

	sum := sha256.Sum256([]byte(raw))
	tokenHash := hex.EncodeToString(sum[:])

	if err := queries.CreatePairingToken(ctx, db.CreatePairingTokenParams{
		TokenHash: tokenHash,
		CreatedBy: sql.NullInt64{Int64: userID, Valid: userID > 0},
	}); err != nil {
		return "", fmt.Errorf("store pairing token: %w", err)
	}

	return raw, nil
}

// ConsumePairingToken looks up a pairing token by its raw value, marks it used
// (one-time-use), and creates a new session for the paired device. Returns the
// session token or an error if the pairing token is invalid, expired, or
// already used.
func ConsumePairingToken(ctx context.Context, queries *db.Queries, rawToken string) (*LoginResult, error) {
	sum := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(sum[:])

	row, err := queries.GetPairingToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("invalid or expired pairing token")
		}
		return nil, fmt.Errorf("lookup pairing token: %w", err)
	}

	// Mark consumed — one-time-use.
	if err := queries.ConsumePairingToken(ctx, row.TokenHash); err != nil {
		return nil, fmt.Errorf("consume pairing token: %w", err)
	}

	// Create a session. Pairing tokens tie back to the user who generated them;
	// if CreatedBy is null (token pre-dates the column) use the first user.
	userID := row.CreatedBy.Int64
	if !row.CreatedBy.Valid || userID == 0 {
		user, err := queries.GetFirstUser(ctx)
		if err != nil {
			return nil, fmt.Errorf("get user for pairing: %w", err)
		}
		userID = user.ID
	}

	sessionToken, err := newSession(ctx, queries, userID)
	if err != nil {
		return nil, err
	}
	return &LoginResult{SessionToken: sessionToken}, nil
}
