package authutil_test

import (
	"context"
	"errors"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/authutil"
)

// TestChangePassword pins what replaces an account's auth key: only the
// current key, a well-formed new one, and a transaction step that succeeds.
// Every refusal leaves the key as it was (#3022).
func TestChangePassword(t *testing.T) {
	ctx := context.Background()
	database := setupWithKeys(t)
	ada := userRow(t, database.Queries, "ada")
	change := func(userID int64, current, next string, after func(db.Querier, int64) error) error {
		_, err := authutil.ChangePassword(ctx, database, authutil.ChangePasswordParams{
			UserID: userID, CurrentAuthKey: current, NewAuthKey: next, AfterChange: after,
		})
		return err
	}
	errStep := errors.New("step failed")

	for _, tc := range []struct {
		name          string
		userID        int64
		current, next string
		after         func(db.Querier, int64) error
		want          error
	}{
		{"wrong key", ada.ID, authKeyOf(2), authKeyOf(3), nil, authutil.ErrIncorrectPassword},
		{"missing key", ada.ID, "", authKeyOf(3), nil, authutil.ErrIncorrectPassword},
		{"raw password", ada.ID, "a-raw-password", authKeyOf(3), nil, authutil.ErrAppTooOld},
		{"malformed new key", ada.ID, authKeyOf(1), "a-raw-password", nil, authutil.ErrInvalidAuthKey},
		{"unknown account", ada.ID + 100, authKeyOf(1), authKeyOf(3), nil, authutil.ErrUserNotFound},
		{"failing step", ada.ID, authKeyOf(1), authKeyOf(3), func(db.Querier, int64) error { return errStep }, errStep},
	} {
		if err := change(tc.userID, tc.current, tc.next, tc.after); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v, want %v", tc.name, err, tc.want)
		}
		if row := userRow(t, database.Queries, "ada"); row.AuthKeyHash != ada.AuthKeyHash {
			t.Fatalf("%s changed the auth key", tc.name)
		}
	}

	stepRan := false
	if err := change(ada.ID, authKeyOf(1), authKeyOf(3), func(_ db.Querier, userID int64) error {
		stepRan = userID == ada.ID
		return nil
	}); err != nil || !stepRan {
		t.Fatalf("change: %v, step ran %v", err, stepRan)
	}
	row := userRow(t, database.Queries, "ada")
	if !authutil.CheckPassword(authKeyOf(3), row.AuthKeyHash) || authutil.CheckPassword(authKeyOf(1), row.AuthKeyHash) {
		t.Error("the account should take the new key and refuse the old one")
	}
	if row.AuthSalt != ada.AuthSalt || row.RecoveryKeyHash != ada.RecoveryKeyHash {
		t.Error("change touched the salt or the recovery key")
	}
}
