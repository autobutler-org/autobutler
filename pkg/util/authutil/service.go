package authutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/autobutler-org/quark/internal/db"
)

// hashToken returns the hex-encoded SHA-256 of a raw session token.
// This digest is what gets stored in the database — the raw token is only
// ever held in memory and returned to the client in the Set-Cookie / response body.
func hashToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(h[:])
}

const (
	// sessionDuration is how far ahead a session's expiry is set, both when it
	// is created and every time it is renewed on use.
	sessionDuration = 30 * 24 * time.Hour // 30 days

	// sessionMaxLifetime caps total session life, measured from creation.
	// Renewal stops here, so a token that leaks dies within this window no
	// matter how actively it is used — the difference between "stay signed in"
	// and "signed in forever".
	sessionMaxLifetime = 90 * 24 * time.Hour // 90 days

	// sessionRenewInterval debounces renewal writes. Renewing on literally
	// every authenticated request would turn each one into a write against
	// SQLite; a session's expiry does not need that resolution.
	sessionRenewInterval = time.Hour
)

// now is the clock used for session expiry. A package var so tests can wind it
// forward instead of sleeping.
var now = time.Now

// renewSession slides a session's expiry forward because it was just used, so
// an actively used session does not expire out from under its user (#1647).
//
// Two bounds keep that from meaning "signed in forever": writes are debounced
// by sessionRenewInterval, and the new expiry is clamped to sessionMaxLifetime
// past creation, after which the session stops renewing and dies on schedule.
//
// GetSession has already rejected an expired session by the time this runs, so
// renewal can never resurrect a dead one.
//
// Errors are logged and swallowed rather than returned: the caller is validly
// authenticated, and failing to extend an expiry is not an authentication
// failure. The session simply keeps the expiry it already had.
func renewSession(ctx context.Context, queries *db.Queries, digest string, session db.GetSessionRow) {
	t := now()
	if t.Sub(session.LastUsedAt) < sessionRenewInterval {
		return
	}

	expiry := t.Add(sessionDuration)
	if limit := session.CreatedAt.Add(sessionMaxLifetime); expiry.After(limit) {
		expiry = limit
	}
	// At the cap the expiry stops moving; skip the write rather than rewriting
	// the same value on every request for the rest of the session's life.
	if !expiry.After(session.ExpiresAt) {
		return
	}

	if err := queries.RenewSession(ctx, db.RenewSessionParams{
		ExpiresAt:  expiry,
		LastUsedAt: t,
		Token:      digest,
	}); err != nil {
		slog.Warn("failed to renew session expiry", "err", err)
	}
}

func newSession(ctx context.Context, queries *db.Queries, userID int64) (string, error) {
	rawToken, err := GenerateSessionToken()
	if err != nil {
		return "", err
	}

	// Store only the hash; the raw token is returned to the caller and set as
	// the cookie/Bearer token. A leaked SQLite database cannot be used to
	// forge valid sessions.
	created := now()
	_, err = queries.CreateSession(ctx, db.CreateSessionParams{
		Token:  hashToken(rawToken),
		UserID: userID,
		// A brand new session counts as just used, so the first renewal is a
		// full sessionRenewInterval away rather than firing immediately.
		ExpiresAt:  created.Add(sessionDuration),
		LastUsedAt: created,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return rawToken, nil
}
