package authutil_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	_ "modernc.org/sqlite"
)

// TestTOTPEnroll_SetsPublicFields verifies that enroll returns a non-empty
// secret and a valid otpauth:// URL.
func TestTOTPEnroll_SetsPublicFields(t *testing.T) {
	queries := newTOTPTestDB(t)
	userID := createTOTPTestUser(t, queries)

	result, err := authutil.TOTPEnroll(context.Background(), queries, userID, "testuser")
	if err != nil {
		t.Fatalf("TOTPEnroll: %v", err)
	}

	if result.Secret == "" {
		t.Error("expected non-empty Secret")
	}
	if len(result.OTPURL) < 10 || result.OTPURL[:len("otpauth://")] != "otpauth://" {
		t.Errorf("expected otpauth:// URL, got %q", result.OTPURL)
	}
}

// TestTOTPIsEnabled_FalseBeforeEnroll verifies that 2FA is not enabled on a
// fresh account.
func TestTOTPIsEnabled_FalseBeforeEnroll(t *testing.T) {
	queries := newTOTPTestDB(t)
	userID := createTOTPTestUser(t, queries)

	enabled, err := authutil.TOTPIsEnabled(context.Background(), queries, userID)
	if err != nil {
		t.Fatalf("TOTPIsEnabled: %v", err)
	}
	if enabled {
		t.Error("expected 2FA to be disabled before enrollment")
	}
}

// TestTOTPConfirm_InvalidCode verifies that a wrong TOTP code is rejected
// during enrollment confirmation.
func TestTOTPConfirm_InvalidCode(t *testing.T) {
	queries := newTOTPTestDB(t)
	userID := createTOTPTestUser(t, queries)

	if _, err := authutil.TOTPEnroll(context.Background(), queries, userID, "testuser"); err != nil {
		t.Fatalf("TOTPEnroll: %v", err)
	}

	err := authutil.TOTPConfirm(context.Background(), queries, userID, "000000")
	if err == nil {
		t.Error("expected error for invalid TOTP code")
	}
}

// TestTOTPConfirm_WithoutEnroll verifies that confirm without a prior enroll
// returns a clear error.
func TestTOTPConfirm_WithoutEnroll(t *testing.T) {
	queries := newTOTPTestDB(t)
	userID := createTOTPTestUser(t, queries)

	err := authutil.TOTPConfirm(context.Background(), queries, userID, "123456")
	if err == nil {
		t.Error("expected error when confirming without prior enroll")
	}
}

// TestTOTPDisable_RemovesSecret verifies that after disabling, TOTPIsEnabled returns false.
func TestTOTPDisable_RemovesSecret(t *testing.T) {
	queries := newTOTPTestDB(t)
	userID := createTOTPTestUser(t, queries)

	// Force-set a secret directly to simulate an enabled state.
	if err := queries.SetTOTPPending(context.Background(), db.SetTOTPPendingParams{
		TotpPending: sql.NullString{String: "BASE32SECRET", Valid: true},
		ID:          userID,
	}); err != nil {
		t.Fatalf("set pending: %v", err)
	}
	if err := queries.ConfirmTOTP(context.Background(), userID); err != nil {
		t.Fatalf("confirm: %v", err)
	}

	enabled, err := authutil.TOTPIsEnabled(context.Background(), queries, userID)
	if err != nil || !enabled {
		t.Fatalf("expected 2FA to be enabled; err=%v enabled=%v", err, enabled)
	}

	if err := authutil.TOTPDisable(context.Background(), queries, userID); err != nil {
		t.Fatalf("TOTPDisable: %v", err)
	}

	enabled, err = authutil.TOTPIsEnabled(context.Background(), queries, userID)
	if err != nil {
		t.Fatalf("TOTPIsEnabled after disable: %v", err)
	}
	if enabled {
		t.Error("expected 2FA to be disabled after calling TOTPDisable")
	}
}

// TestIssueTOTPChallenge_Roundtrip verifies that a challenge can be consumed.
func TestIssueTOTPChallenge_Roundtrip(t *testing.T) {
	queries := newTOTPTestDB(t)
	userID := createTOTPTestUser(t, queries)

	rawToken, err := authutil.IssueTOTPChallenge(context.Background(), queries, userID)
	if err != nil {
		t.Fatalf("IssueTOTPChallenge: %v", err)
	}
	if rawToken == "" {
		t.Fatal("expected non-empty token")
	}

	gotUserID, err := authutil.ConsumeTOTPChallenge(context.Background(), queries, rawToken)
	if err != nil {
		t.Fatalf("ConsumeTOTPChallenge: %v", err)
	}
	if gotUserID != userID {
		t.Errorf("userID mismatch: got %d, want %d", gotUserID, userID)
	}
}

// TestIssueTOTPChallenge_OneTimeUse verifies that a challenge token cannot be
// consumed twice.
func TestIssueTOTPChallenge_OneTimeUse(t *testing.T) {
	queries := newTOTPTestDB(t)
	userID := createTOTPTestUser(t, queries)

	rawToken, err := authutil.IssueTOTPChallenge(context.Background(), queries, userID)
	if err != nil {
		t.Fatalf("IssueTOTPChallenge: %v", err)
	}

	// First consumption succeeds.
	if _, err := authutil.ConsumeTOTPChallenge(context.Background(), queries, rawToken); err != nil {
		t.Fatalf("first consume: %v", err)
	}

	// Second consumption must fail.
	if _, err := authutil.ConsumeTOTPChallenge(context.Background(), queries, rawToken); err == nil {
		t.Error("expected error on second consume (one-time use)")
	}
}

// TestConsumeTOTPChallenge_Invalid verifies that an unknown token is rejected.
func TestConsumeTOTPChallenge_Invalid(t *testing.T) {
	queries := newTOTPTestDB(t)

	_, err := authutil.ConsumeTOTPChallenge(context.Background(), queries, "notavalidtoken")
	if err == nil {
		t.Error("expected error for unknown challenge token")
	}
}

// TestLoginOrChallenge_NoTOTP verifies that a user without 2FA gets a direct
// LoginResult (no challenge).
func TestLoginOrChallenge_NoTOTP(t *testing.T) {
	queries := newTOTPTestDB(t)

	setupResult, err := authutil.Setup(context.Background(), queries, authutil.SetupParams{
		Username: "alice",
		Password: "supersecure",
	})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if setupResult.SessionToken == "" {
		t.Fatal("expected session token from setup")
	}

	result, challenge, err := authutil.LoginOrChallenge(context.Background(), queries, authutil.LoginParams{
		Username: "alice",
		Password: "supersecure",
	})
	if err != nil {
		t.Fatalf("LoginOrChallenge: %v", err)
	}
	if challenge != nil {
		t.Error("expected no 2FA challenge for user without TOTP enabled")
	}
	if result == nil || result.SessionToken == "" {
		t.Error("expected a session token")
	}
}

// TestLoginOrChallenge_WithTOTP verifies that a user with 2FA enabled gets a
// challenge, not a direct session.
func TestLoginOrChallenge_WithTOTP(t *testing.T) {
	queries := newTOTPTestDB(t)

	if _, err := authutil.Setup(context.Background(), queries, authutil.SetupParams{
		Username: "bob",
		Password: "supersecure",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	user, err := queries.GetUserByUsername(context.Background(), "bob")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	// Force-enable TOTP.
	if err := queries.SetTOTPPending(context.Background(), db.SetTOTPPendingParams{
		TotpPending: sql.NullString{String: "BASE32SECRET", Valid: true},
		ID:          user.ID,
	}); err != nil {
		t.Fatalf("set pending: %v", err)
	}
	if err := queries.ConfirmTOTP(context.Background(), user.ID); err != nil {
		t.Fatalf("confirm: %v", err)
	}

	result, challenge, err := authutil.LoginOrChallenge(context.Background(), queries, authutil.LoginParams{
		Username: "bob",
		Password: "supersecure",
	})
	if err != nil {
		t.Fatalf("LoginOrChallenge: %v", err)
	}
	if result != nil {
		t.Error("expected no direct session for user with TOTP enabled")
	}
	if challenge == nil {
		t.Fatal("expected a 2FA challenge")
	}
	if !challenge.Requires2FA {
		t.Error("challenge.Requires2FA should be true")
	}
	if challenge.ChallengeToken == "" {
		t.Error("challenge.ChallengeToken should be non-empty")
	}
}
