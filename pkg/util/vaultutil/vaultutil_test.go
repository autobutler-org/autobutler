package vaultutil_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
)

// keyOf is a well-formed vault key whose bytes are all fill.
func keyOf(fill byte) vaultutil.Key {
	b := func(n int) []byte { return bytes.Repeat([]byte{fill}, n) }
	return vaultutil.Key{
		WrappedByPassword: b(72), SaltPw: b(16),
		WrappedByPhrase: b(72), SaltRp: b(16),
		KdfParams: json.RawMessage(`{"alg":"argon2id13","opsLimit":3,"memLimit":67108864}`),
	}
}

func itemOf(fill byte) vaultutil.ItemFields {
	return vaultutil.ItemFields{Ciphertext: bytes.Repeat([]byte{fill}, 40), Nonce: bytes.Repeat([]byte{fill}, 24), FormatVersion: 1}
}

// onEachBackend runs test against a SQLite database, and against PostgreSQL
// under `make test/db/postgres`, each with two accounts, bob and carol.
func onEachBackend(t *testing.T, test func(t *testing.T, database *db.DatabaseSqlc, bob, carol int64)) {
	t.Run("sqlite", func(t *testing.T) {
		database := dbtest.NewDB(t)
		bob, carol := addAccounts(t, database)
		test(t, database, bob, carol)
	})
	t.Run("postgres", func(t *testing.T) {
		database, err := db.ConnectToPostgres(dbtest.PostgresURL(t))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = database.Db.Close() })
		bob, carol := addAccounts(t, database)
		test(t, database, bob, carol)
	})
}

// addAccounts adds bob and carol and returns their ids.
func addAccounts(t *testing.T, database *db.DatabaseSqlc) (int64, int64) {
	t.Helper()
	ids := make([]int64, 0, 2)
	for _, name := range []string{"bob", "carol"} {
		user, err := database.Queries.CreateUser(context.Background(), db.CreateUserParams{Username: name, PasswordHash: "h", RecoveryPhraseHash: "r"})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, user.ID)
	}
	return ids[0], ids[1]
}

func TestValidateKey(t *testing.T) {
	for name, edit := range map[string]func(*vaultutil.Key){
		"no password wrap":      func(k *vaultutil.Key) { k.WrappedByPassword = nil },
		"oversized wrap":        func(k *vaultutil.Key) { k.WrappedByPassword = make([]byte, vaultutil.MaxWrappedKeyBytes+1) },
		"short salt":            func(k *vaultutil.Key) { k.SaltPw = k.SaltPw[:15] },
		"phrase wrap alone":     func(k *vaultutil.Key) { k.SaltRp = nil },
		"phrase salt alone":     func(k *vaultutil.Key) { k.WrappedByPhrase = nil },
		"empty phrase wrap":     func(k *vaultutil.Key) { k.WrappedByPhrase = []byte{} },
		"kdf params not object": func(k *vaultutil.Key) { k.KdfParams = json.RawMessage(`[1]`) },
		"no kdf params":         func(k *vaultutil.Key) { k.KdfParams = nil },
	} {
		key := keyOf(1)
		edit(&key)
		if err := vaultutil.ValidateKey(key); !errors.Is(err, vaultutil.ErrInvalidKey) {
			t.Errorf("%s: ValidateKey = %v, want ErrInvalidKey", name, err)
		}
	}
	withoutPhrase := keyOf(1)
	withoutPhrase.WrappedByPhrase, withoutPhrase.SaltRp = nil, nil
	for _, key := range []vaultutil.Key{keyOf(1), withoutPhrase} {
		if err := vaultutil.ValidateKey(key); err != nil {
			t.Errorf("ValidateKey(%v) = %v, want nil", key, err)
		}
	}
}

func TestValidateItem(t *testing.T) {
	for name, edit := range map[string]func(*vaultutil.ItemFields){
		"no ciphertext":     func(f *vaultutil.ItemFields) { f.Ciphertext = nil },
		"no nonce":          func(f *vaultutil.ItemFields) { f.Nonce = nil },
		"oversized nonce":   func(f *vaultutil.ItemFields) { f.Nonce = make([]byte, vaultutil.MaxNonceBytes+1) },
		"no format version": func(f *vaultutil.ItemFields) { f.FormatVersion = 0 },
	} {
		fields := itemOf(1)
		edit(&fields)
		if err := vaultutil.ValidateItem(fields); !errors.Is(err, vaultutil.ErrInvalidItem) {
			t.Errorf("%s: ValidateItem = %v, want ErrInvalidItem", name, err)
		}
	}
}

// TestKey creates, re-wraps and reads an account's vault key, and never lets
// a second create replace it.
func TestKey(t *testing.T) {
	onEachBackend(t, testKey)
}

func testKey(t *testing.T, database *db.DatabaseSqlc, bob, carol int64) {
	ctx := context.Background()
	q := database.Queries

	if _, err := vaultutil.GetKey(vaultutil.GetKeyParams{Ctx: ctx, Queries: q, UserID: bob}); !errors.Is(err, vaultutil.ErrKeyNotFound) {
		t.Fatalf("GetKey before setup = %v, want ErrKeyNotFound", err)
	}
	invalid := keyOf(1)
	invalid.SaltPw = nil
	if _, err := vaultutil.CreateKey(vaultutil.CreateKeyParams{Ctx: ctx, Queries: q, UserID: bob, Key: invalid}); !errors.Is(err, vaultutil.ErrInvalidKey) {
		t.Errorf("CreateKey with a bad key = %v, want ErrInvalidKey", err)
	}
	created, err := vaultutil.CreateKey(vaultutil.CreateKeyParams{Ctx: ctx, Queries: q, UserID: bob, Key: keyOf(1)})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(created.Key.WrappedByPassword, keyOf(1).WrappedByPassword) || string(created.Key.KdfParams) != string(keyOf(1).KdfParams) {
		t.Errorf("CreateKey stored %+v", created.Key)
	}
	if _, err := vaultutil.CreateKey(vaultutil.CreateKeyParams{Ctx: ctx, Queries: q, UserID: bob, Key: keyOf(2)}); !errors.Is(err, vaultutil.ErrKeyExists) {
		t.Errorf("second CreateKey = %v, want ErrKeyExists", err)
	}

	// A credential change re-wraps the key bob has, and gives carol, who has
	// none, her first.
	for _, user := range []int64{bob, carol} {
		if _, err := vaultutil.PutKey(vaultutil.PutKeyParams{Ctx: ctx, Queries: q, UserID: user, Key: keyOf(3)}); err != nil {
			t.Fatal(err)
		}
		got, err := vaultutil.GetKey(vaultutil.GetKeyParams{Ctx: ctx, Queries: q, UserID: user})
		if err != nil || !bytes.Equal(got.Key.WrappedByPassword, keyOf(3).WrappedByPassword) {
			t.Errorf("after PutKey user %d has %+v (%v), want the re-wrapped key", user, got.Key, err)
		}
	}
	if _, err := vaultutil.PutKey(vaultutil.PutKeyParams{Ctx: ctx, Queries: q, UserID: bob, Key: invalid}); !errors.Is(err, vaultutil.ErrInvalidKey) {
		t.Errorf("PutKey with a bad key = %v, want ErrInvalidKey", err)
	}
}

// TestItems drives an account's items, and checks another account reads
// every one of them as not found.
func TestItems(t *testing.T) {
	onEachBackend(t, testItems)
}

func testItems(t *testing.T, database *db.DatabaseSqlc, bob, carol int64) {
	ctx := context.Background()
	q := database.Queries

	if _, err := vaultutil.CreateItem(vaultutil.CreateItemParams{Ctx: ctx, Database: database, UserID: bob, Fields: itemOf(1)}); !errors.Is(err, vaultutil.ErrKeyNotFound) {
		t.Fatalf("CreateItem with no vault = %v, want ErrKeyNotFound", err)
	}
	if _, err := vaultutil.CreateKey(vaultutil.CreateKeyParams{Ctx: ctx, Queries: q, UserID: bob, Key: keyOf(1)}); err != nil {
		t.Fatal(err)
	}
	if _, err := vaultutil.CreateItem(vaultutil.CreateItemParams{Ctx: ctx, Database: database, UserID: bob, Fields: vaultutil.ItemFields{}}); !errors.Is(err, vaultutil.ErrInvalidItem) {
		t.Errorf("CreateItem with an empty item = %v, want ErrInvalidItem", err)
	}
	first, err := vaultutil.CreateItem(vaultutil.CreateItemParams{Ctx: ctx, Database: database, UserID: bob, Fields: itemOf(1)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vaultutil.CreateItem(vaultutil.CreateItemParams{Ctx: ctx, Database: database, UserID: bob, Fields: itemOf(2)}); err != nil {
		t.Fatal(err)
	}
	id := first.Item.ID

	listed, err := vaultutil.ListItems(vaultutil.ListItemsParams{Ctx: ctx, Queries: q, UserID: bob})
	if err != nil || len(listed.Items) != 2 || listed.Items[0].ID != id {
		t.Fatalf("ListItems = %+v (%v), want bob's two items oldest first", listed.Items, err)
	}
	if theirs, err := vaultutil.ListItems(vaultutil.ListItemsParams{Ctx: ctx, Queries: q, UserID: carol}); err != nil || theirs.Items == nil || len(theirs.Items) != 0 {
		t.Errorf("carol's ListItems = %#v (%v), want an empty, non-nil list", theirs.Items, err)
	}

	updated, err := vaultutil.UpdateItem(vaultutil.UpdateItemParams{Ctx: ctx, Queries: q, UserID: bob, ID: id, Fields: itemOf(3)})
	if err != nil || !bytes.Equal(updated.Item.Ciphertext, itemOf(3).Ciphertext) {
		t.Errorf("UpdateItem = %+v (%v)", updated.Item, err)
	}
	if _, err := vaultutil.UpdateItem(vaultutil.UpdateItemParams{Ctx: ctx, Queries: q, UserID: bob, ID: id, Fields: vaultutil.ItemFields{}}); !errors.Is(err, vaultutil.ErrInvalidItem) {
		t.Errorf("UpdateItem with an empty item = %v, want ErrInvalidItem", err)
	}

	// Carol reaches none of bob's items, and changes none.
	if _, err := vaultutil.GetItem(vaultutil.GetItemParams{Ctx: ctx, Queries: q, UserID: carol, ID: id}); !errors.Is(err, vaultutil.ErrItemNotFound) {
		t.Errorf("carol GetItem = %v, want ErrItemNotFound", err)
	}
	if _, err := vaultutil.UpdateItem(vaultutil.UpdateItemParams{Ctx: ctx, Queries: q, UserID: carol, ID: id, Fields: itemOf(4)}); !errors.Is(err, vaultutil.ErrItemNotFound) {
		t.Errorf("carol UpdateItem = %v, want ErrItemNotFound", err)
	}
	if _, err := vaultutil.DeleteItem(vaultutil.DeleteItemParams{Ctx: ctx, Queries: q, UserID: carol, ID: id}); !errors.Is(err, vaultutil.ErrItemNotFound) {
		t.Errorf("carol DeleteItem = %v, want ErrItemNotFound", err)
	}
	got, err := vaultutil.GetItem(vaultutil.GetItemParams{Ctx: ctx, Queries: q, UserID: bob, ID: id})
	if err != nil || !bytes.Equal(got.Item.Ciphertext, itemOf(3).Ciphertext) {
		t.Errorf("bob GetItem after carol's attempts = %+v (%v)", got.Item, err)
	}

	if _, err := vaultutil.DeleteItem(vaultutil.DeleteItemParams{Ctx: ctx, Queries: q, UserID: bob, ID: id}); err != nil {
		t.Fatal(err)
	}
	if _, err := vaultutil.DeleteItem(vaultutil.DeleteItemParams{Ctx: ctx, Queries: q, UserID: bob, ID: id}); !errors.Is(err, vaultutil.ErrItemNotFound) {
		t.Errorf("second DeleteItem = %v, want ErrItemNotFound", err)
	}
}

// TestDeleteVault removes one account's items and key and leaves another's.
func TestDeleteVault(t *testing.T) {
	onEachBackend(t, testDeleteVault)
}

func testDeleteVault(t *testing.T, database *db.DatabaseSqlc, bob, carol int64) {
	ctx := context.Background()
	q := database.Queries
	for _, user := range []int64{bob, carol} {
		if _, err := vaultutil.CreateKey(vaultutil.CreateKeyParams{Ctx: ctx, Queries: q, UserID: user, Key: keyOf(1)}); err != nil {
			t.Fatal(err)
		}
		if _, err := vaultutil.CreateItem(vaultutil.CreateItemParams{Ctx: ctx, Database: database, UserID: user, Fields: itemOf(1)}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := vaultutil.DeleteVault(vaultutil.DeleteVaultParams{Ctx: ctx, Database: database, UserID: bob}); err != nil {
		t.Fatal(err)
	}
	if _, err := vaultutil.GetKey(vaultutil.GetKeyParams{Ctx: ctx, Queries: q, UserID: bob}); !errors.Is(err, vaultutil.ErrKeyNotFound) {
		t.Errorf("bob's key after DeleteVault: %v, want ErrKeyNotFound", err)
	}
	if listed, _ := vaultutil.ListItems(vaultutil.ListItemsParams{Ctx: ctx, Queries: q, UserID: bob}); len(listed.Items) != 0 {
		t.Errorf("bob has %d items after DeleteVault, want 0", len(listed.Items))
	}
	if listed, _ := vaultutil.ListItems(vaultutil.ListItemsParams{Ctx: ctx, Queries: q, UserID: carol}); len(listed.Items) != 1 {
		t.Errorf("carol has %d items after bob's DeleteVault, want 1", len(listed.Items))
	}
	// Starting over is allowed, and deleting nothing is not an error.
	if _, err := vaultutil.DeleteVault(vaultutil.DeleteVaultParams{Ctx: ctx, Database: database, UserID: bob}); err != nil {
		t.Errorf("DeleteVault with no vault = %v", err)
	}
	if _, err := vaultutil.CreateKey(vaultutil.CreateKeyParams{Ctx: ctx, Queries: q, UserID: bob, Key: keyOf(2)}); err != nil {
		t.Errorf("CreateKey after DeleteVault = %v", err)
	}
}

func TestPublishChanged(t *testing.T) {
	bus := eventbus.New()
	events, unsubscribe := bus.Subscribe("vaultutil-test")
	defer unsubscribe()
	vaultutil.PublishChanged(vaultutil.PublishChangedParams{EventBus: bus, UserID: 4, ItemID: 9})
	vaultutil.PublishChanged(vaultutil.PublishChangedParams{UserID: 4})
	evt := <-events
	if data, ok := evt.Data.(eventbus.VaultChanged); evt.Kind != eventbus.EventVaultChanged || !ok || data != (eventbus.VaultChanged{ItemID: 9, UserID: 4}) {
		t.Errorf("published %+v, want vault_changed for item 9 of user 4", evt)
	}
}

func TestCreateItemConcurrent(t *testing.T) {
	onEachBackend(t, testCreateItemConcurrent)
}

// testCreateItemConcurrent creates items from many goroutines at once. On
// SQLite a transaction that reads before it writes fails with SQLITE_BUSY the
// moment another writer commits in between, which was a 500 on a vault item
// create that happened to land beside any other write.
func testCreateItemConcurrent(t *testing.T, database *db.DatabaseSqlc, bob, carol int64) {
	ctx := context.Background()
	for _, user := range []int64{bob, carol} {
		if _, err := vaultutil.CreateKey(vaultutil.CreateKeyParams{Ctx: ctx, Queries: database.Queries, UserID: user, Key: keyOf(1)}); err != nil {
			t.Fatal(err)
		}
	}
	const writers = 16
	errs := make(chan error, writers*5)
	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		user := bob
		if w%2 == 1 {
			user = carol
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 5 {
				_, err := vaultutil.CreateItem(vaultutil.CreateItemParams{Ctx: ctx, Database: database, UserID: user, Fields: itemOf(1)})
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("CreateItem beside other writers: %v", err)
		}
	}
}
