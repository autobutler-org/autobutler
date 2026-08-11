package authutil

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	totpIssuer         = "AutoButler"
	challengeTokenSize = 32 // bytes → 64 hex chars
)

// TOTPEnrollResult is returned when a user begins TOTP enrollment.
type TOTPEnrollResult struct {
	// Secret is the base32 TOTP secret (for manual entry in authenticator apps).
	Secret string
	// OTPURL is the otpauth:// URI — pass to a QR code generator.
	OTPURL string
}

// TOTPEnroll generates a new TOTP secret for the user and stores it as pending.
// The user must call TOTPConfirm with a valid code to activate 2FA.
func TOTPEnroll(ctx context.Context, queries *db.Queries, userID int64, username string) (*TOTPEnrollResult, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: username,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("totp generate: %w", err)
	}

	if err := queries.SetTOTPPending(ctx, db.SetTOTPPendingParams{
		TotpPending: sql.NullString{String: key.Secret(), Valid: true},
		ID:          userID,
	}); err != nil {
		return nil, fmt.Errorf("store pending totp: %w", err)
	}

	return &TOTPEnrollResult{
		Secret: key.Secret(),
		OTPURL: key.URL(),
	}, nil
}

// TOTPConfirm validates a TOTP code against the pending secret and activates 2FA.
func TOTPConfirm(ctx context.Context, queries *db.Queries, userID int64, code string) error {
	pending, err := queries.GetTOTPPending(ctx, userID)
	if err != nil {
		return fmt.Errorf("get pending totp: %w", err)
	}
	if !pending.Valid || pending.String == "" {
		return fmt.Errorf("no pending TOTP enrollment found — call enroll first")
	}

	valid, err := totp.ValidateCustom(code, pending.String, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1, // ±1 window tolerance (90 seconds total)
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil || !valid {
		return fmt.Errorf("invalid TOTP code")
	}

	return queries.ConfirmTOTP(ctx, userID)
}

// TOTPDisable removes TOTP from the user's account.
// Caller is responsible for verifying the current password before calling this.
func TOTPDisable(ctx context.Context, queries *db.Queries, userID int64) error {
	return queries.DisableTOTP(ctx, userID)
}

// TOTPIsEnabled returns true if the user has an active TOTP secret.
func TOTPIsEnabled(ctx context.Context, queries *db.Queries, userID int64) (bool, error) {
	secret, err := queries.GetTOTPSecret(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("get totp secret: %w", err)
	}
	return secret.Valid && secret.String != "", nil
}

// TOTPVerify checks a TOTP code against the user's active secret.
func TOTPVerify(ctx context.Context, queries *db.Queries, userID int64, code string) error {
	secret, err := queries.GetTOTPSecret(ctx, userID)
	if err != nil {
		return fmt.Errorf("get totp secret: %w", err)
	}
	if !secret.Valid || secret.String == "" {
		return fmt.Errorf("2FA is not enabled for this account")
	}

	valid, err := totp.ValidateCustom(code, secret.String, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil || !valid {
		return fmt.Errorf("invalid TOTP code")
	}
	return nil
}

// IssueTOTPChallenge creates a short-lived challenge token for the 2FA verify step.
// Returns the raw (unhashed) token that should be sent to the client.
func IssueTOTPChallenge(ctx context.Context, queries *db.Queries, userID int64) (string, error) {
	// Reuse GenerateSessionToken; challengeTokenSize == sessionTokenSize.
	// If sizes diverge in the future, call rand.Read directly.
	raw, err := GenerateSessionToken()
	if err != nil {
		return "", fmt.Errorf("generate challenge token: %w", err)
	}

	// Purge expired challenges to keep the table small.
	_ = queries.PurgeTOTPChallenges(ctx)

	tokenHash := hashToken(raw)
	if err := queries.CreateTOTPChallenge(ctx, db.CreateTOTPChallengeParams{
		UserID:    userID,
		TokenHash: tokenHash,
	}); err != nil {
		return "", fmt.Errorf("store challenge: %w", err)
	}
	return raw, nil
}

// ConsumeTOTPChallenge looks up and deletes a challenge token (one-time use).
// Returns the userID the challenge belongs to, or an error if not found/expired.
func ConsumeTOTPChallenge(ctx context.Context, queries *db.Queries, rawToken string) (int64, error) {
	tokenHash := hashToken(rawToken)
	challenge, err := queries.GetTOTPChallenge(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("invalid or expired 2FA challenge token")
		}
		return 0, fmt.Errorf("lookup challenge: %w", err)
	}

	// Consume immediately — one-time use.
	_ = queries.DeleteTOTPChallenge(ctx, tokenHash)

	return challenge.UserID, nil
}

// hashToken lives in service.go — shared by session and challenge tokens.

// VerifyTOTPChallenge completes the 2FA login flow: it consumes a challenge
// token issued by LoginOrChallenge, validates the supplied TOTP code against
// that user's secret, and issues a session on success.
//
// The challenge is consumed even when the code turns out to be wrong, so a
// stolen challenge token cannot be used to brute-force codes — the caller must
// re-authenticate with their password to obtain a fresh one.
func VerifyTOTPChallenge(ctx context.Context, queries *db.Queries, challengeToken, code string) (*LoginResult, error) {
	userID, err := ConsumeTOTPChallenge(ctx, queries, challengeToken)
	if err != nil {
		return nil, err
	}

	if err := TOTPVerify(ctx, queries, userID, code); err != nil {
		return nil, err
	}

	token, err := newSession(ctx, queries, userID)
	if err != nil {
		return nil, err
	}

	return &LoginResult{SessionToken: token}, nil
}
