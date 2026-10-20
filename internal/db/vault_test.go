package db_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/sqlutil"
)

// TestVaultQueriesStayInsideTheAccount checks every per-account vault query
// (#3458) finds, changes and deletes only the rows of the user_id it is given,
// and that a vault key is set up once and then only re-wrapped.
func TestVaultQueriesStayInsideTheAccount(t *testing.T) {
	database := dbtest.NewDB(t)
	q := database.Queries
	ctx := context.Background()
	var alice, bob int64
	for name, id := range map[string]*int64{"alice": &alice, "bob": &bob} {
		if err := database.Db.QueryRow(`INSERT INTO users (username, password_hash, recovery_phrase_hash) VALUES (?, '', '') RETURNING id`, name).Scan(id); err != nil {
			t.Fatal(err)
		}
	}

	key := db.CreateVaultKeyParams{UserID: alice, WrappedByPassword: []byte("pw"), SaltPw: []byte("salt"), KdfParams: "{}"}
	if _, err := q.CreateVaultKey(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, err := q.CreateVaultKey(ctx, key); !sqlutil.IsUniqueConstraintErr(err) {
		t.Fatalf("second CreateVaultKey = %v, want a unique violation", err)
	}
	if _, err := q.GetVaultKey(ctx, bob); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("bob's GetVaultKey = %v, want no rows", err)
	}
	rewrap := db.UpdateVaultKeyParams{
		UserID: bob, WrappedByPassword: []byte("pw2"), SaltPw: []byte("salt"),
		WrappedByPhrase: []byte("rp"), SaltRp: []byte("salt"), KdfParams: "{}",
	}
	if n, err := q.UpdateVaultKey(ctx, rewrap); err != nil || n != 0 {
		t.Fatalf("UpdateVaultKey for an account with no vault = %d, %v", n, err)
	}
	rewrap.UserID = alice
	if n, err := q.UpdateVaultKey(ctx, rewrap); err != nil || n != 1 {
		t.Fatalf("UpdateVaultKey = %d, %v", n, err)
	}
	got, err := q.GetVaultKey(ctx, alice)
	if err != nil || string(got.WrappedByPassword) != "pw2" || string(got.WrappedByPhrase) != "rp" {
		t.Fatalf("GetVaultKey after re-wrap = %+v, %v", got, err)
	}

	item, err := q.CreateVaultItem(ctx, db.CreateVaultItemParams{UserID: alice, Ciphertext: []byte("c"), Nonce: []byte("n"), FormatVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.GetVaultItem(ctx, db.GetVaultItemParams{ID: item.ID, UserID: bob}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("bob's GetVaultItem = %v, want no rows", err)
	}
	if _, err := q.UpdateVaultItem(ctx, db.UpdateVaultItemParams{ID: item.ID, UserID: bob, Ciphertext: []byte("x"), Nonce: []byte("x"), FormatVersion: 1}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("bob's UpdateVaultItem = %v, want no rows", err)
	}
	if n, err := q.DeleteVaultItem(ctx, db.DeleteVaultItemParams{ID: item.ID, UserID: bob}); err != nil || n != 0 {
		t.Fatalf("bob's DeleteVaultItem = %d, %v", n, err)
	}
	if items, err := q.ListVaultItems(ctx, bob); err != nil || len(items) != 0 {
		t.Fatalf("bob's ListVaultItems = %v, %v", items, err)
	}
	if err := q.DeleteVaultItems(ctx, bob); err != nil {
		t.Fatal(err)
	}

	updated, err := q.UpdateVaultItem(ctx, db.UpdateVaultItemParams{ID: item.ID, UserID: alice, Ciphertext: []byte("c2"), Nonce: []byte("n2"), FormatVersion: 2})
	if err != nil || string(updated.Ciphertext) != "c2" || updated.FormatVersion != 2 {
		t.Fatalf("UpdateVaultItem = %+v, %v", updated, err)
	}
	if items, err := q.ListVaultItems(ctx, alice); err != nil || len(items) != 1 {
		t.Fatalf("ListVaultItems = %v, %v", items, err)
	}
	if n, err := q.DeleteVaultItem(ctx, db.DeleteVaultItemParams{ID: item.ID, UserID: alice}); err != nil || n != 1 {
		t.Fatalf("DeleteVaultItem = %d, %v", n, err)
	}
}
