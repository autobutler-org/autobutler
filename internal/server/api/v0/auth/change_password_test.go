package v0_auth_test

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/chatutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

// TestChangePassword drives PUT /auth/password for a member: refused without
// a session or the current password's auth key, all-or-nothing with chat
// keys, and afterwards the new password signs in and the old one does not,
// while the sessions the account already had stay signed in (#3022).
func TestChangePassword(t *testing.T) {
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
	login := func(key string) int {
		return postJSON(engine, "/api/v0/auth/login", map[string]string{"username": "bob", "authKey": key}).Code
	}
	session, _ := decodeBody(t, postJSON(engine, "/api/v0/auth/login", map[string]string{"username": "bob", "authKey": oldKey}))["token"].(string)
	// unchanged fails unless bob still has the password and chat keys he
	// started with.
	unchanged := func(after string) {
		t.Helper()
		if row := userByName(t, queries, "bob"); row.AuthKeyHash != bob.AuthKeyHash {
			t.Errorf("%s changed the auth key", after)
		}
		got, err := chatutil.GetKeys(chatutil.GetKeysParams{Ctx: ctx, Queries: queries, UserID: bob.ID})
		if err != nil || !bytes.Equal(got.Keys.WrappedByPassword, chatKeysOf(1).WrappedByPassword) {
			t.Errorf("%s changed the chat keys: %v", after, err)
		}
	}

	if w := putAs(t, database, db.User{}, path, map[string]any{"password": oldKey, "newAuthKey": newKey}); w.Code != http.StatusUnauthorized {
		t.Errorf("without a session = %d, want 401", w.Code)
	}

	// A session alone cannot change the password: a wrong or missing current
	// key gets delete-account's 403, a raw password the update-the-app 426.
	for name, body := range map[string]map[string]any{
		"wrong key":   {"password": authKeyOf(2), "newAuthKey": newKey, "chatKeys": chatKeysOf(3)},
		"missing key": {"newAuthKey": newKey, "chatKeys": chatKeysOf(3)},
	} {
		if w := putAs(t, database, bob, path, body); w.Code != http.StatusForbidden || decodeBody(t, w)["error"] != authutil.ErrIncorrectPassword.Error() {
			t.Errorf("%s = %d %s, want 403 incorrect password", name, w.Code, w.Body)
		}
	}
	raw := putAs(t, database, bob, path, map[string]any{"password": "original-password", "newAuthKey": newKey, "chatKeys": chatKeysOf(3)})
	if raw.Code != http.StatusUpgradeRequired || decodeBody(t, raw)["error"] != authutil.ErrAppTooOld.Error() {
		t.Errorf("raw password = %d %s, want 426 with the update-the-app copy", raw.Code, raw.Body)
	}
	unchanged("a refused re-confirmation")

	for name, body := range map[string]map[string]any{
		"raw new password":    {"password": oldKey, "newAuthKey": "new-password"},
		"no new key":          {"password": oldKey},
		"malformed chat keys": {"password": oldKey, "newAuthKey": newKey, "chatKeys": chatutil.Keys{}},
		"oversized body":      {"password": oldKey, "newAuthKey": newKey, "padding": strings.Repeat("x", chatutil.MaxKeysRequestBytes)},
	} {
		if w := putAs(t, database, bob, path, body); w.Code != http.StatusBadRequest {
			t.Errorf("%s = %d %s, want 400", name, w.Code, w.Body)
		}
	}
	unchanged("a malformed body")

	// A chat key write that fails inside the transaction rolls the new auth
	// key back with it.
	if _, err := database.Db.Exec(`CREATE TRIGGER refuse_chat_keys BEFORE INSERT ON user_chat_keys BEGIN SELECT RAISE(ABORT, 'refused'); END`); err != nil {
		t.Fatal(err)
	}
	if w := putAs(t, database, bob, path, map[string]any{"password": oldKey, "newAuthKey": newKey, "chatKeys": chatKeysOf(3)}); w.Code != http.StatusInternalServerError {
		t.Errorf("failing chat keys write = %d %s, want 500", w.Code, w.Body)
	}
	if _, err := database.Db.Exec(`DROP TRIGGER refuse_chat_keys`); err != nil {
		t.Fatal(err)
	}
	unchanged("a failed chat keys write")
	if code := login(oldKey); code != http.StatusOK {
		t.Errorf("login with the old password after a failed change = %d, want 200", code)
	}

	if w := putAs(t, database, bob, path, map[string]any{"password": oldKey, "newAuthKey": newKey, "chatKeys": chatKeysOf(3)}); w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Fatalf("change = %d %s, want 204 and no body", w.Code, w.Body)
	}
	got, err := chatutil.GetKeys(chatutil.GetKeysParams{Ctx: ctx, Queries: queries, UserID: bob.ID})
	if err != nil || !bytes.Equal(got.Keys.WrappedByPassword, chatKeysOf(3).WrappedByPassword) {
		t.Errorf("change didn't store the re-wrapped chat keys: %v", err)
	}
	if row := userByName(t, queries, "bob"); row.AuthSalt != bob.AuthSalt {
		t.Error("change replaced the salt the client derived the new key with")
	}
	if code := login(oldKey); code != http.StatusUnauthorized {
		t.Errorf("login with the old password = %d, want 401", code)
	}
	if code := login(newKey); code != http.StatusOK {
		t.Errorf("login with the new password = %d, want 200", code)
	}
	if _, _, err := authutil.ValidateSession(ctx, queries, session); err != nil {
		t.Errorf("a session from before the change was signed out: %v", err)
	}
	// The old key no longer re-confirms the caller either.
	if w := putAs(t, database, bob, path, map[string]any{"password": oldKey, "newAuthKey": authKeyOf(7)}); w.Code != http.StatusForbidden {
		t.Errorf("a second change with the old key = %d, want 403", w.Code)
	}

	// Without chatKeys the stored ones are left as they are.
	if w := putAs(t, database, bob, path, map[string]any{"password": newKey, "newAuthKey": authKeyOf(7)}); w.Code != http.StatusNoContent {
		t.Fatalf("change without chat keys = %d %s, want 204", w.Code, w.Body)
	}
	if got, err := chatutil.GetKeys(chatutil.GetKeysParams{Ctx: ctx, Queries: queries, UserID: bob.ID}); err != nil || !bytes.Equal(got.Keys.WrappedByPassword, chatKeysOf(3).WrappedByPassword) {
		t.Errorf("change without chat keys touched the stored ones: %v", err)
	}

	// An account still on its password has no auth key to re-confirm with: it
	// moves to one by signing in.
	if w := putAs(t, database, userByName(t, queries, "old"), path, map[string]any{"password": dbtest.AuthKey("old-password"), "newAuthKey": newKey}); w.Code != http.StatusForbidden {
		t.Errorf("legacy account = %d %s, want 403", w.Code, w.Body)
	}
}
