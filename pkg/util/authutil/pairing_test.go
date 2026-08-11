package authutil_test

import (
	"context"
	"testing"

	"github.com/autobutler-org/autobutler/pkg/util/authutil"
)

// TestCreatePairingToken_ReturnsNonEmptyToken verifies that a fresh pairing
// token is non-empty.
func TestCreatePairingToken_ReturnsNonEmptyToken(t *testing.T) {
	queries := newTOTPTestDB(t) // reuses the schema helper from totp_test.go
	userID := createTOTPTestUser(t, queries)

	token, err := authutil.CreatePairingToken(context.Background(), queries, userID)
	if err != nil {
		t.Fatalf("CreatePairingToken: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

// TestConsumePairingToken_Roundtrip verifies that consuming a valid token
// returns a LoginResult with a non-empty session token.
func TestConsumePairingToken_Roundtrip(t *testing.T) {
	queries := newTOTPTestDB(t)
	// Setup creates a user via authutil.Setup so newSession can find the user.
	if _, err := authutil.Setup(context.Background(), queries, authutil.SetupParams{
		Username: "pairuser",
		Password: "password123",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	user, err := queries.GetUserByUsername(context.Background(), "pairuser")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	token, err := authutil.CreatePairingToken(context.Background(), queries, user.ID)
	if err != nil {
		t.Fatalf("CreatePairingToken: %v", err)
	}

	result, err := authutil.ConsumePairingToken(context.Background(), queries, token)
	if err != nil {
		t.Fatalf("ConsumePairingToken: %v", err)
	}
	if result == nil || result.SessionToken == "" {
		t.Error("expected non-empty session token")
	}
}

// TestConsumePairingToken_OneTimeUse verifies that consuming the same token
// twice fails on the second attempt.
func TestConsumePairingToken_OneTimeUse(t *testing.T) {
	queries := newTOTPTestDB(t)
	if _, err := authutil.Setup(context.Background(), queries, authutil.SetupParams{
		Username: "pairuser2",
		Password: "password123",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}

	user, err := queries.GetUserByUsername(context.Background(), "pairuser2")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	token, err := authutil.CreatePairingToken(context.Background(), queries, user.ID)
	if err != nil {
		t.Fatalf("CreatePairingToken: %v", err)
	}

	// First consume succeeds.
	if _, err := authutil.ConsumePairingToken(context.Background(), queries, token); err != nil {
		t.Fatalf("first consume: %v", err)
	}

	// Second consume must fail.
	if _, err := authutil.ConsumePairingToken(context.Background(), queries, token); err == nil {
		t.Error("expected error on second consume (one-time-use)")
	}
}

// TestConsumePairingToken_InvalidToken verifies that an unknown token is rejected.
func TestConsumePairingToken_InvalidToken(t *testing.T) {
	queries := newTOTPTestDB(t)
	_, err := authutil.ConsumePairingToken(context.Background(), queries, "notavalidtoken")
	if err == nil {
		t.Error("expected error for unknown pairing token")
	}
}
