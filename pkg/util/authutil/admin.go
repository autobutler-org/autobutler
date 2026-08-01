package authutil

import (
	"context"
	"errors"
	"fmt"

	"github.com/autobutler-org/autobutler/internal/db"
)

// IsAdmin returns true if the given username has the admin role.
// Returns false (not an error) for unknown users.
func IsAdmin(ctx context.Context, queries *db.Queries, username string) (bool, error) {
	val, err := queries.IsUserAdmin(ctx, username)
	if err != nil {
		return false, fmt.Errorf("check admin: %w", err)
	}
	return val != 0, nil
}

// PromoteToAdmin grants admin to the given username.
func PromoteToAdmin(ctx context.Context, queries *db.Queries, username string) error {
	if _, err := queries.PromoteToAdmin(ctx, username); err != nil {
		return fmt.Errorf("promote %q to admin: %w", username, err)
	}
	return nil
}

// DemoteFromAdmin removes admin from the given username, refusing if they are
// the last admin.
func DemoteFromAdmin(ctx context.Context, queries *db.Queries, username string) error {
	count, err := queries.GetAdminCount(ctx)
	if err != nil {
		return fmt.Errorf("count admins: %w", err)
	}
	if count <= 1 {
		return errors.New("cannot demote the last admin")
	}
	if _, err := queries.DemoteFromAdmin(ctx, username); err != nil {
		return fmt.Errorf("demote %q: %w", username, err)
	}
	return nil
}
