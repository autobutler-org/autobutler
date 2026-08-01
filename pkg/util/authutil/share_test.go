package authutil_test

import (
	"context"
	"testing"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/authutil"
)

func TestCreateShareLink_ReturnsNonEmptyToken(t *testing.T) {
	queries := newTOTPTestDB(t)
	userID := createTOTPTestUser(t, queries)

	tok, err := authutil.CreateShareLink(context.Background(), queries, authutil.ShareLinkParams{
		UserID:       userID,
		ResourceType: "file",
		ResourcePath: "photos/grandma.jpg",
		DeviceSerial: "",
	})
	if err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}
	if len(tok) < 16 {
		t.Errorf("expected non-trivial token, got %q", tok)
	}
}

func TestValidateShareLink_ValidToken(t *testing.T) {
	queries := newTOTPTestDB(t)
	userID := createTOTPTestUser(t, queries)

	tok, err := authutil.CreateShareLink(context.Background(), queries, authutil.ShareLinkParams{
		UserID:       userID,
		ResourceType: "folder",
		ResourcePath: "2024/vacation",
		DeviceSerial: "abc123",
	})
	if err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}

	link, err := authutil.ValidateShareLink(context.Background(), queries, tok)
	if err != nil {
		t.Fatalf("ValidateShareLink: %v", err)
	}
	if link.ResourcePath != "2024/vacation" {
		t.Errorf("expected ResourcePath %q, got %q", "2024/vacation", link.ResourcePath)
	}
	if link.DeviceSerial != "abc123" {
		t.Errorf("expected DeviceSerial %q, got %q", "abc123", link.DeviceSerial)
	}
}

func TestValidateShareLink_InvalidToken(t *testing.T) {
	queries := newTOTPTestDB(t)

	_, err := authutil.ValidateShareLink(context.Background(), queries, "totallyinvalidtoken")
	if err == nil {
		t.Error("expected error for invalid token, got nil")
	}
}

func TestValidateShareLink_ExpiredToken(t *testing.T) {
	queries := newTOTPTestDB(t)
	userID := createTOTPTestUser(t, queries)

	past := time.Now().UTC().Add(-1 * time.Hour)
	tok, err := authutil.CreateShareLink(context.Background(), queries, authutil.ShareLinkParams{
		UserID:       userID,
		ResourceType: "file",
		ResourcePath: "old.pdf",
		ExpiresAt:    &past, // already expired
	})
	if err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}

	_, err = authutil.ValidateShareLink(context.Background(), queries, tok)
	if err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestRevokeShareLink_Roundtrip(t *testing.T) {
	queries := newTOTPTestDB(t)
	userID := createTOTPTestUser(t, queries)

	tok, err := authutil.CreateShareLink(context.Background(), queries, authutil.ShareLinkParams{
		UserID:       userID,
		ResourceType: "file",
		ResourcePath: "doc.pdf",
	})
	if err != nil {
		t.Fatalf("CreateShareLink: %v", err)
	}

	// Validate — should succeed.
	link, err := authutil.ValidateShareLink(context.Background(), queries, tok)
	if err != nil {
		t.Fatalf("ValidateShareLink before revoke: %v", err)
	}

	// Revoke using the token hash (as the management API would).
	if err := authutil.RevokeShareLink(context.Background(), queries, tok, userID); err != nil {
		t.Fatalf("RevokeShareLink: %v", err)
	}
	_ = link

	// Validate again — should fail.
	_, err = authutil.ValidateShareLink(context.Background(), queries, tok)
	if err == nil {
		t.Error("expected error after revoke, got nil")
	}
}
