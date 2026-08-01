package authutil_test

import (
	"context"
	"testing"
	"time"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/authutil"
)

func newAdminTestDB(t *testing.T) (*db.Queries, int64) {
	t.Helper()
	queries := newTOTPTestDB(t)
	userID := createTOTPTestUser(t, queries)
	return queries, userID
}

func TestSetup_FirstUserIsAdmin(t *testing.T) {
	queries := newTOTPTestDB(t)
	_, err := authutil.Setup(context.Background(), queries, authutil.SetupParams{
		Username: "firstuser",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}

	isAdmin, err := authutil.IsAdmin(context.Background(), queries, "firstuser")
	if err != nil {
		t.Fatalf("IsAdmin: %v", err)
	}
	if !isAdmin {
		t.Error("expected first user to be admin after Setup")
	}
}

func TestIsAdmin_FalseForNonAdmin(t *testing.T) {
	queries, _ := newAdminTestDB(t)
	// createTOTPTestUser creates "testuser" without admin flag
	isAdmin, err := authutil.IsAdmin(context.Background(), queries, "testuser")
	if err != nil {
		t.Fatalf("IsAdmin: %v", err)
	}
	if isAdmin {
		t.Error("expected testuser not to be admin")
	}
}

func TestPromoteToAdmin(t *testing.T) {
	queries, _ := newAdminTestDB(t)

	if err := authutil.PromoteToAdmin(context.Background(), queries, "testuser"); err != nil {
		t.Fatalf("PromoteToAdmin: %v", err)
	}

	isAdmin, err := authutil.IsAdmin(context.Background(), queries, "testuser")
	if err != nil {
		t.Fatalf("IsAdmin after promote: %v", err)
	}
	if !isAdmin {
		t.Error("expected testuser to be admin after promote")
	}
}

func TestDemoteFromAdmin_AllowedWhenMultipleAdmins(t *testing.T) {
	queries, _ := newAdminTestDB(t)

	// Create a second user and make both admins.
	hash, _ := authutil.HashPassword("pass")
	second, err := queries.CreateUser(context.Background(), db.CreateUserParams{
		Username: "seconduser", PasswordHash: hash, RecoveryPhraseHash: hash,
	})
	if err != nil {
		t.Fatalf("create second user: %v", err)
	}
	_ = second

	_ = authutil.PromoteToAdmin(context.Background(), queries, "testuser")
	_ = authutil.PromoteToAdmin(context.Background(), queries, "seconduser")

	// Demoting one should succeed.
	if err := authutil.DemoteFromAdmin(context.Background(), queries, "testuser"); err != nil {
		t.Fatalf("DemoteFromAdmin: %v", err)
	}

	isAdmin, _ := authutil.IsAdmin(context.Background(), queries, "testuser")
	if isAdmin {
		t.Error("expected testuser not to be admin after demote")
	}
}

func TestDemoteFromAdmin_RefusesLastAdmin(t *testing.T) {
	queries, _ := newAdminTestDB(t)

	// Make testuser the sole admin.
	_ = authutil.PromoteToAdmin(context.Background(), queries, "testuser")

	err := authutil.DemoteFromAdmin(context.Background(), queries, "testuser")
	if err == nil {
		t.Error("expected error when demoting last admin, got nil")
	}
}

// Unused var to avoid "imported and not used" on time in tests that don't need it.
var _ = time.Now
