package authutil

import (
	"context"
	"fmt"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
)

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
func ValidateSession(ctx context.Context, queries *db.Queries, token string) (string, error) {
	session, err := queries.GetSession(ctx, token)
	if err != nil {
		return "", fmt.Errorf("invalid or expired session")
	}
	return session.Username, nil
}

// Logout deletes a session token.
func Logout(ctx context.Context, queries *db.Queries, token string) error {
	return queries.DeleteSession(ctx, token)
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
	token, err := GenerateSessionToken()
	if err != nil {
		return "", err
	}

	_, err = queries.CreateSession(ctx, db.CreateSessionParams{
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(sessionDuration),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return token, nil
}
