package v0_auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	v0_auth "github.com/autobutler-org/quark/internal/server/api/v0/auth"
	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/chatutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

// vaultKeyOf is a well-formed wrapped vault key whose bytes are all fill.
func vaultKeyOf(fill byte) vaultutil.Key {
	b := func(n int) []byte { return bytes.Repeat([]byte{fill}, n) }
	return vaultutil.Key{
		WrappedByPassword: b(72), SaltPw: b(16),
		WrappedByPhrase: b(72), SaltRp: b(16),
		KdfParams: json.RawMessage(`{"alg":"argon2id13","opsLimit":3,"memLimit":67108864}`),
	}
}

// vaultPasswordWrap is the password wrap of userID's vault key, nil when it
// has none.
func vaultPasswordWrap(t *testing.T, queries db.Querier, userID int64) []byte {
	t.Helper()
	got, err := vaultutil.GetKey(vaultutil.GetKeyParams{Ctx: context.Background(), Queries: queries, UserID: userID})
	if err != nil {
		return nil
	}
	return got.Key.WrappedByPassword
}

// TestChangePassword_VaultKey re-wraps the vault key with the password, in the
// one transaction: a malformed vault key or a failed write changes nothing,
// the chat keys and the password included.
func TestChangePassword_VaultKey(t *testing.T) {
	const path = "/api/v0/auth/password"
	ctx := context.Background()
	engine, database := newAuthKeyEngine(t)
	queries := database.Queries
	oldKey, newKey := dbtest.AuthKey("original-password"), dbtest.AuthKey("new-password")
	if _, err := authutil.CreateUser(ctx, authutil.CreateUserParams{Database: database, Username: "bob", AuthKey: oldKey, SaltSecret: dbtest.SaltSecret, Files: vfs.NewMemVFS("files")}); err != nil {
		t.Fatal(err)
	}
	bob := userByName(t, queries, "bob")
	if _, err := chatutil.PutKeys(chatutil.PutKeysParams{Ctx: ctx, Queries: queries, UserID: bob.ID, Keys: chatKeysOf(1)}); err != nil {
		t.Fatal(err)
	}
	if _, err := vaultutil.CreateKey(vaultutil.CreateKeyParams{Ctx: ctx, Queries: queries, UserID: bob.ID, Key: vaultKeyOf(1)}); err != nil {
		t.Fatal(err)
	}
	unchanged := func(after string) {
		t.Helper()
		if row := userByName(t, queries, "bob"); row.AuthKeyHash != bob.AuthKeyHash {
			t.Errorf("%s changed the auth key", after)
		}
		if got, _ := chatutil.GetKeys(chatutil.GetKeysParams{Ctx: ctx, Queries: queries, UserID: bob.ID}); !bytes.Equal(got.Keys.WrappedByPassword, chatKeysOf(1).WrappedByPassword) {
			t.Errorf("%s changed the chat keys", after)
		}
		if !bytes.Equal(vaultPasswordWrap(t, queries, bob.ID), vaultKeyOf(1).WrappedByPassword) {
			t.Errorf("%s changed the vault key", after)
		}
	}

	bad := vaultKeyOf(3)
	bad.SaltRp = nil
	if w := putAs(t, database, bob, path, map[string]any{"password": oldKey, "newAuthKey": newKey, "chatKeys": chatKeysOf(3), "vaultKey": bad}); w.Code != http.StatusBadRequest {
		t.Errorf("malformed vault key = %d %s, want 400", w.Code, w.Body)
	}
	unchanged("a malformed vault key")

	if _, err := database.Db.Exec(`CREATE TRIGGER refuse_vault_keys BEFORE UPDATE ON vault_keys BEGIN SELECT RAISE(ABORT, 'refused'); END`); err != nil {
		t.Fatal(err)
	}
	if w := putAs(t, database, bob, path, map[string]any{"password": oldKey, "newAuthKey": newKey, "chatKeys": chatKeysOf(3), "vaultKey": vaultKeyOf(3)}); w.Code != http.StatusInternalServerError {
		t.Errorf("failing vault key write = %d %s, want 500", w.Code, w.Body)
	}
	if _, err := database.Db.Exec(`DROP TRIGGER refuse_vault_keys`); err != nil {
		t.Fatal(err)
	}
	unchanged("a failed vault key write")

	if w := putAs(t, database, bob, path, map[string]any{"password": oldKey, "newAuthKey": newKey, "chatKeys": chatKeysOf(3), "vaultKey": vaultKeyOf(3)}); w.Code != http.StatusNoContent {
		t.Fatalf("change = %d %s, want 204", w.Code, w.Body)
	}
	if !bytes.Equal(vaultPasswordWrap(t, queries, bob.ID), vaultKeyOf(3).WrappedByPassword) {
		t.Error("change didn't store the re-wrapped vault key")
	}
	if code := postJSON(engine, "/api/v0/auth/login", map[string]string{"username": "bob", "authKey": newKey}).Code; code != http.StatusOK {
		t.Errorf("login with the new password = %d, want 200", code)
	}

	// Without vaultKey the stored one is left as it is.
	if w := putAs(t, database, bob, path, map[string]any{"password": newKey, "newAuthKey": authKeyOf(7)}); w.Code != http.StatusNoContent {
		t.Fatalf("change without a vault key = %d %s, want 204", w.Code, w.Body)
	}
	if !bytes.Equal(vaultPasswordWrap(t, queries, bob.ID), vaultKeyOf(3).WrappedByPassword) {
		t.Error("change without a vault key touched the stored one")
	}
}

// TestSetRecoveryKey_VaultKey re-wraps the vault key with a new recovery key,
// in the one transaction.
func TestSetRecoveryKey_VaultKey(t *testing.T) {
	ctx := context.Background()
	_, database := newAuthKeyEngine(t)
	queries := database.Queries
	key := dbtest.AuthKey("original-password")
	if _, err := authutil.CreateUser(ctx, authutil.CreateUserParams{Database: database, Username: "bob", AuthKey: key, SaltSecret: dbtest.SaltSecret, Files: vfs.NewMemVFS("files")}); err != nil {
		t.Fatal(err)
	}
	bob := userByName(t, queries, "bob")
	if _, err := vaultutil.CreateKey(vaultutil.CreateKeyParams{Ctx: ctx, Queries: queries, UserID: bob.ID, Key: vaultKeyOf(1)}); err != nil {
		t.Fatal(err)
	}

	if w := putRecoveryKey(t, database, bob, map[string]any{"password": key, "recoveryKey": authKeyOf(9), "vaultKey": vaultutil.Key{}}); w.Code != http.StatusBadRequest {
		t.Errorf("malformed vault key = %d %s, want 400", w.Code, w.Body)
	}
	if row := userByName(t, queries, "bob"); row.RecoveryKeyHash != "" {
		t.Error("a malformed vault key let the recovery key through")
	}

	if w := putRecoveryKey(t, database, bob, map[string]any{"password": key, "recoveryKey": authKeyOf(9), "vaultKey": vaultKeyOf(4)}); w.Code != http.StatusNoContent {
		t.Fatalf("rotation = %d %s, want 204", w.Code, w.Body)
	}
	if row := userByName(t, queries, "bob"); row.RecoveryKeyHash == "" {
		t.Error("rotation should store the recovery key")
	}
	if !bytes.Equal(vaultPasswordWrap(t, queries, bob.ID), vaultKeyOf(4).WrappedByPassword) {
		t.Error("rotation didn't store the re-wrapped vault key")
	}
}

// TestRecover_VaultKey drives the vault's half of recovery: fetch the wrapped
// vault key with the recovery key, as /auth/recover/keys does for chat, then
// reset the password and store the re-wrapped key in one request, announced
// to the account alone.
func TestRecover_VaultKey(t *testing.T) {
	database := dbtest.NewDB(t)
	ctx := context.Background()
	if _, err := authutil.Setup(ctx, authutil.SetupParams{Database: database, Files: vfs.NewMemVFS("files"), Username: "admin", AuthKey: dbtest.AuthKey("admin-password"), SaltSecret: dbtest.SaltSecret}); err != nil {
		t.Fatal(err)
	}
	const phrase = "apple-bread-cloud-delta-eagle-flame"
	createRecoverableUser(t, database.Queries, "bob", phrase)
	bob := userByName(t, database.Queries, "bob")

	// newPublicAuthEngine resets the settings the salt secret lives in; this
	// engine carries an event bus besides.
	newPublicAuthEngine(t, database)
	bus := eventbus.New()
	events, unsubscribe := bus.Subscribe("auth-vault-key-test")
	t.Cleanup(unsubscribe)
	deps := deputil.NewDependencies().WithDatabase(database).WithEventBus(bus)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c = ctxutil.With(c, "deps", deps)
		c.Next()
	})
	serverutil.RegisterRouterWithGroup(engine.Group("/api/v0"), v0_auth.NewRouter())
	post := func(path string, body any) *httptest.ResponseRecorder {
		return postJSON(engine, "/api/v0"+path, body)
	}
	fetch := map[string]string{"username": "bob", "recoveryKey": dbtest.AuthKey(phrase)}
	recoverWith := func(newPassword string, vaultKey any) *httptest.ResponseRecorder {
		return post("/auth/recover", map[string]any{"username": "bob", "recoveryKey": dbtest.AuthKey(phrase), "newAuthKey": dbtest.AuthKey(newPassword), "vaultKey": vaultKey})
	}

	if w := post("/auth/recover/vault-key", fetch); w.Code != http.StatusNotFound {
		t.Fatalf("fetch before bob has a vault = %d %s, want 404", w.Code, w.Body)
	}
	if _, err := vaultutil.CreateKey(vaultutil.CreateKeyParams{Ctx: ctx, Queries: database.Queries, UserID: bob.ID, Key: vaultKeyOf(1)}); err != nil {
		t.Fatal(err)
	}
	if w := post("/auth/recover/vault-key", map[string]string{"username": "bob", "recoveryKey": dbtest.AuthKey("wrong-phrase")}); w.Code != http.StatusBadRequest {
		t.Errorf("fetch with a wrong phrase = %d, want 400", w.Code)
	}
	if w := post("/auth/recover/vault-key", map[string]string{"username": "nobody", "recoveryKey": dbtest.AuthKey(phrase)}); w.Code != http.StatusBadRequest {
		t.Errorf("fetch for an unknown account = %d, want 400", w.Code)
	}
	w := post("/auth/recover/vault-key", fetch)
	if w.Code != http.StatusOK {
		t.Fatalf("fetch = %d %s, want 200", w.Code, w.Body)
	}
	var fetched vaultutil.Key
	if err := json.Unmarshal(w.Body.Bytes(), &fetched); err != nil || !bytes.Equal(fetched.WrappedByPhrase, vaultKeyOf(1).WrappedByPhrase) {
		t.Errorf("fetched = %s, want bob's phrase wrap", w.Body)
	}

	// A malformed vault key refuses the whole recovery: the old password
	// still works and the key is as it was.
	if w := recoverWith("brand-new-password", vaultutil.Key{SaltPw: []byte{1}}); w.Code != http.StatusBadRequest {
		t.Fatalf("recover with a malformed vault key = %d %s, want 400", w.Code, w.Body)
	}
	if _, err := authutil.Login(ctx, database.Queries, authutil.LoginParams{Username: "bob", AuthKey: dbtest.AuthKey("brand-new-password")}); err == nil {
		t.Error("a refused recovery changed the password")
	}
	if !bytes.Equal(vaultPasswordWrap(t, database.Queries, bob.ID), vaultKeyOf(1).WrappedByPassword) {
		t.Error("a refused recovery changed the vault key")
	}

	for len(events) > 0 {
		<-events
	}
	if w := recoverWith("brand-new-password", vaultKeyOf(5)); w.Code != http.StatusOK {
		t.Fatalf("recover = %d %s, want 200", w.Code, w.Body)
	}
	if !bytes.Equal(vaultPasswordWrap(t, database.Queries, bob.ID), vaultKeyOf(5).WrappedByPassword) {
		t.Error("recover didn't store the re-wrapped vault key")
	}
	select {
	case evt := <-events:
		if evt.Kind != eventbus.EventVaultChanged || evt.Data != (eventbus.VaultChanged{UserID: bob.ID}) {
			t.Errorf("recover published %+v, want vault_changed for bob", evt)
		}
	default:
		t.Error("recover with a vault key published nothing")
	}
}
