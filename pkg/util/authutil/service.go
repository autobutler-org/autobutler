package authutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

const sessionDuration = 30 * 24 * time.Hour // 30 days

// SetupParams contains parameters for first-boot user setup.
type SetupParams struct {
	Username string
	Password string
}

// SetupResult contains the result of first-boot setup.
type SetupResult struct {
	SessionToken   string
	RecoveryPhrase string // shown once — caller must surface to user
}

// LoginParams contains parameters for login.
type LoginParams struct {
	Username string
	Password string
}

// LoginResult contains the result of a successful login.
type LoginResult struct {
	SessionToken string
}

// RecoverParams contains parameters for password recovery.
type RecoverParams struct {
	RecoveryPhrase string
	NewPassword    string
}

// IsSetupComplete returns true if at least one user exists.
func IsSetupComplete(ctx context.Context, queries *db.Queries) (bool, error) {
	count, err := queries.CountUsers(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to count users: %w", err)
	}
	return count > 0, nil
}

// Setup creates the first user and returns a session token + recovery phrase.
// Returns an error if setup has already been completed.
func Setup(ctx context.Context, queries *db.Queries, params SetupParams) (*SetupResult, error) {
	if params.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if len(params.Password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	complete, err := IsSetupComplete(ctx, queries)
	if err != nil {
		return nil, err
	}
	if complete {
		return nil, fmt.Errorf("setup already complete")
	}

	passwordHash, err := HashPassword(params.Password)
	if err != nil {
		return nil, err
	}

	recoveryPhrase, err := GenerateRecoveryPhrase()
	if err != nil {
		return nil, err
	}

	recoveryHash, err := HashPassword(recoveryPhrase)
	if err != nil {
		return nil, err
	}

	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Username:           params.Username,
		PasswordHash:       passwordHash,
		RecoveryPhraseHash: recoveryHash,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// First user is automatically the admin.
	if err := queries.SetUserAdmin(ctx, db.SetUserAdminParams{
		IsAdmin:  1,
		Username: user.Username,
	}); err != nil {
		return nil, fmt.Errorf("promote first user to admin: %w", err)
	}

	token, err := newSession(ctx, queries, user.ID)
	if err != nil {
		return nil, err
	}

	return &SetupResult{
		SessionToken:   token,
		RecoveryPhrase: recoveryPhrase,
	}, nil
}

// Login validates credentials and returns a session token.
func Login(ctx context.Context, queries *db.Queries, params LoginParams) (*LoginResult, error) {
	user, err := queries.GetUserByUsername(ctx, params.Username)
	if err != nil {
		// Don't leak whether the username exists
		return nil, fmt.Errorf("invalid credentials")
	}

	if !CheckPassword(params.Password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	token, err := newSession(ctx, queries, user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResult{SessionToken: token}, nil
}

// ValidateSession checks a session token and returns the username if valid.
// The raw token is hashed before the DB lookup — tokens are never stored plaintext.
func ValidateSession(ctx context.Context, queries *db.Queries, token string) (string, error) {
	session, err := queries.GetSession(ctx, hashToken(token))
	if err != nil {
		return "", fmt.Errorf("invalid or expired session")
	}
	return session.Username, nil
}

// ValidateBasicAuth checks a username/password pair against the user database.
// Returns the username if valid, or an error if not. Unlike Login, this does
// not create a session — each request authenticates independently.
func ValidateBasicAuth(ctx context.Context, queries *db.Queries, username, password string) (string, error) {
	user, err := queries.GetUserByUsername(ctx, username)
	if err != nil {
		return "", fmt.Errorf("invalid credentials")
	}
	if !CheckPassword(password, user.PasswordHash) {
		return "", fmt.Errorf("invalid credentials")
	}
	return user.Username, nil
}

// Logout deletes a session token.
func Logout(ctx context.Context, queries *db.Queries, token string) error {
	return queries.DeleteSession(ctx, hashToken(token))
}

// Recover resets a user's password using their recovery phrase.
func Recover(ctx context.Context, queries *db.Queries, params RecoverParams) (*LoginResult, error) {
	if len(params.NewPassword) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	// We need to check the recovery phrase against all users (there's only one in single-user mode)
	// Get all users — for now just check the first/only user
	count, err := queries.CountUsers(ctx)
	if err != nil || count == 0 {
		return nil, fmt.Errorf("invalid recovery phrase")
	}

	// TODO(#350): multi-user recovery needs to match the recovery phrase to a
	// specific user. For now, single-user mode — find the first (only) user.
	user, err := queries.GetFirstUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("invalid recovery phrase")
	}

	normalized := NormalizeRecoveryPhrase(params.RecoveryPhrase)
	if !CheckPassword(normalized, user.RecoveryPhraseHash) {
		return nil, fmt.Errorf("invalid recovery phrase")
	}

	newHash, err := HashPassword(params.NewPassword)
	if err != nil {
		return nil, err
	}

	if err := queries.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           user.ID,
		PasswordHash: newHash,
	}); err != nil {
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	// Invalidate all existing sessions for this user
	if err := queries.DeleteUserSessions(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("failed to invalidate sessions: %w", err)
	}

	token, err := newSession(ctx, queries, user.ID)
	if err != nil {
		return nil, err
	}

	return &LoginResult{SessionToken: token}, nil
}

func newSession(ctx context.Context, queries *db.Queries, userID int64) (string, error) {
	rawToken, err := GenerateSessionToken()
	if err != nil {
		return "", err
	}

	// Store only the hash; the raw token is returned to the caller and set as
	// the cookie/Bearer token. A leaked SQLite database cannot be used to
	// forge valid sessions.
	_, err = queries.CreateSession(ctx, db.CreateSessionParams{
		Token:     hashToken(rawToken),
		UserID:    userID,
		ExpiresAt: time.Now().Add(sessionDuration),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return rawToken, nil
}

// SessionInfo is a safe, token-free representation of an active session
// returned to the caller. The ID is the hex-encoded SHA-256 of the raw token
// so clients can reference a session for revocation without exposing the token.
type SessionInfo struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// ListActiveSessions returns all non-expired sessions for the given user.
// Because tokens are stored as SHA-256 digests, the digest itself is used
// as the opaque session ID exposed to clients.
func ListActiveSessions(ctx context.Context, queries *db.Queries, userID int64) ([]SessionInfo, error) {
	rows, err := queries.ListActiveSessionsForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	out := make([]SessionInfo, 0, len(rows))
	for _, s := range rows {
		out = append(out, SessionInfo{
			// s.Token is already the SHA-256 digest (stored that way in newSession).
			ID:        s.Token,
			CreatedAt: s.CreatedAt,
			ExpiresAt: s.ExpiresAt,
		})
	}
	return out, nil
}

// RevokeSession deletes the session whose stored digest matches the given id,
// scoped to the given user. Returns true if a session was deleted, false if
// no matching session was found.
func RevokeSession(ctx context.Context, queries *db.Queries, userID int64, id string) (bool, error) {
	rows, err := queries.ListActiveSessionsForUser(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to list sessions: %w", err)
	}
	for _, s := range rows {
		// s.Token is the stored digest; id is also a digest.
		if s.Token == id {
			if err := queries.DeleteSession(ctx, s.Token); err != nil {
				return false, fmt.Errorf("failed to delete session: %w", err)
			}
			return true, nil
		}
	}
	return false, nil
}

// RevokeAllSessions deletes all sessions for the given user.
func RevokeAllSessions(ctx context.Context, queries *db.Queries, userID int64) error {
	if err := queries.DeleteUserSessions(ctx, userID); err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}
	return nil
}

// PurgeExpiredSessions removes all sessions whose expiry has passed. (#1330)
// GetSession already guards against expired tokens, so this is a housekeeping
// operation rather than a security gate.
func PurgeExpiredSessions(ctx context.Context, queries *db.Queries) error {
	if err := queries.DeleteExpiredSessions(ctx); err != nil {
		return fmt.Errorf("purge expired sessions: %w", err)
	}
	return nil
}
