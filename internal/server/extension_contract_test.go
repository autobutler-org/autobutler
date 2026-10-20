package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/internal/server/middleware"
	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/settingsutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

// TestExtensionContract pins the request and response shapes the browser
// extension (autobutler-org/quark-browser-extensions, src/shared/api.ts) depends on.
// That client lives in another repository, so nothing else here would notice
// a renamed field or a changed status code. A change that breaks this test
// needs a matching change in the extension.
//
// The vault half is the per-account vault of #3459: the extension fetches the
// account's wrapped vault key, opens it on its own, and reads and writes items
// the Quark stores as ciphertext.
func TestExtensionContract(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	settingsutil.ResetForTesting(filepath.Join(t.TempDir(), "settings.json"))
	t.Cleanup(func() { settingsutil.ResetForTesting("") })
	ctx := context.Background()

	database := dbtest.NewDB(t)
	if _, err := authutil.Setup(ctx, authutil.SetupParams{Database: database, Files: vfs.NewMemVFS("files"), Username: "admin", AuthKey: dbtest.AuthKey("admin-password"), SaltSecret: dbtest.SaltSecret}); err != nil {
		t.Fatalf("authutil.Setup: %v", err)
	}
	for _, account := range []struct{ name, status string }{{"pending", authutil.StatusPending}, {"member", authutil.StatusActive}} {
		hash, err := authutil.HashKey(dbtest.AuthKey(account.name + "-password"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := database.Queries.CreateUser(ctx, db.CreateUserParams{Username: account.name, AuthKeyHash: hash}); err != nil {
			t.Fatal(err)
		}
		if account.status == authutil.StatusActive {
			continue
		}
		if _, err := database.Queries.SetUserStatus(ctx, db.SetUserStatusParams{ToStatus: account.status, Username: account.name, FromStatus: authutil.StatusActive}); err != nil {
			t.Fatal(err)
		}
	}

	deps := deputil.NewDependencies().WithDatabase(database)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	middleware.Use(engine, deps)
	setupRouters(engine, nil, deps)
	// Requests leave connected-device records running; they land before the
	// database closes (#2772).
	t.Cleanup(deps.Background().Wait)

	do := func(method, path, token string, body any) (int, any) {
		t.Helper()
		var reader io.Reader
		if body != nil {
			encoded, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			reader = bytes.NewReader(encoded)
		}
		req := httptest.NewRequest(method, path, reader)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)
		if w.Code >= http.StatusInternalServerError {
			t.Logf("%s %s: status %d: %s", method, path, w.Code, w.Body.String())
		}
		var decoded any
		if w.Body.Len() > 0 {
			if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
				t.Fatalf("%s %s: body is not JSON: %s", method, path, w.Body.String())
			}
		}
		return w.Code, decoded
	}
	// doObject is do for a response whose body is a JSON object.
	doObject := func(method, path, token string, body any) (int, map[string]any) {
		t.Helper()
		code, decoded := do(method, path, token, body)
		object, _ := decoded.(map[string]any)
		return code, object
	}

	// The extension signs in with an auth key (#2430): the salt, then the key
	// the client derived with it. It must ship that before this Quark does,
	// because the raw password alone, which it used to send, is now a 426
	// whose error is a sentence to show the user. The password beside the key
	// is the one-time upgrade of a legacy account, and an account that has a
	// key is checked by it.
	code, body := doObject(http.MethodGet, "/api/v0/auth/salt?username=admin", "", nil)
	expectStatus(t, "salt", code, http.StatusOK)
	expectString(t, "salt", body, "salt")
	expectBool(t, "salt legacy", body, "legacy")
	expectBool(t, "salt legacyRecovery", body, "legacyRecovery")

	authKey := dbtest.AuthKey("admin-password")
	code, body = doObject(http.MethodPost, "/api/v0/auth/login", "", map[string]string{"username": "admin", "authKey": authKey})
	expectStatus(t, "login with an auth key", code, http.StatusOK)
	token := expectString(t, "auth key login token", body, "token")

	code, body = doObject(http.MethodPost, "/api/v0/auth/login", "", map[string]string{"username": "admin", "password": "admin-password"})
	expectStatus(t, "login with the raw password", code, http.StatusUpgradeRequired)
	expectString(t, "raw password error", body, "error")

	code, _ = doObject(http.MethodPost, "/api/v0/auth/login", "", map[string]string{"username": "admin", "password": "admin-password", "authKey": authKey})
	expectStatus(t, "login with the password beside the auth key", code, http.StatusOK)

	code, _ = doObject(http.MethodPost, "/api/v0/auth/login", "", map[string]string{"username": "admin", "authKey": dbtest.AuthKey("wrong")})
	expectStatus(t, "login with a wrong auth key", code, http.StatusUnauthorized)

	code, body = doObject(http.MethodPost, "/api/v0/auth/login", "", map[string]string{"username": "pending", "authKey": dbtest.AuthKey("pending-password")})
	expectStatus(t, "login to a pending account", code, http.StatusForbidden)
	if status := expectString(t, "refusal status", body, "status"); status != authutil.StatusPending {
		t.Errorf("refusal status = %q, want %q", status, authutil.StatusPending)
	}
	expectString(t, "refusal error", body, "error")

	code, body = doObject(http.MethodPost, "/api/v0/auth/login", "", map[string]string{"username": "member", "authKey": dbtest.AuthKey("member-password")})
	expectStatus(t, "member login", code, http.StatusOK)
	memberToken := expectString(t, "member login token", body, "token")

	code, body = doObject(http.MethodGet, "/api/v0/vault/key", "", nil)
	expectStatus(t, "vault key without a token", code, http.StatusUnauthorized)
	if msg := expectString(t, "missing-token error", body, "error"); msg != "authentication required" {
		t.Errorf("missing-token error = %q; the extension tells an expired session from everything else by this text", msg)
	}

	// An account with no vault: 404 for the key, and items cannot be added.
	code, body = doObject(http.MethodGet, "/api/v0/vault/key", token, nil)
	expectStatus(t, "vault key before setup", code, http.StatusNotFound)
	expectString(t, "no vault error", body, "error")
	item := func(fill byte) map[string]any {
		return map[string]any{"ciphertext": b64(fill, 48), "nonce": b64(fill, 24), "formatVersion": 1}
	}
	code, body = doObject(http.MethodPost, "/api/v0/vault/items", token, item(1))
	expectStatus(t, "create an item before setup", code, http.StatusConflict)
	expectString(t, "no vault on create error", body, "error")

	// Setting the vault up re-confirms the login password's auth key, so the
	// key is never wrapped under anything else (#3461).
	key := map[string]any{
		"password":          dbtest.AuthKey("wrong"),
		"wrappedByPassword": b64(1, 72), "saltPw": b64(1, 16),
		"wrappedByPhrase": b64(2, 72), "saltRp": b64(2, 16),
		"kdfParams": map[string]any{"alg": "argon2id13", "opsLimit": 3, "memLimit": 67108864},
	}
	code, body = doObject(http.MethodPut, "/api/v0/vault/key", token, key)
	expectStatus(t, "set the vault up with a wrong password", code, http.StatusForbidden)
	expectString(t, "wrong password error", body, "error")
	key["password"] = authKey
	code, body = doObject(http.MethodPut, "/api/v0/vault/key", token, key)
	expectStatus(t, "set the vault up", code, http.StatusCreated)
	expectKey(t, "created key", body, key)

	code, body = doObject(http.MethodPut, "/api/v0/vault/key", token, key)
	expectStatus(t, "set the vault up again", code, http.StatusConflict)
	expectString(t, "vault exists error", body, "error")

	code, body = doObject(http.MethodGet, "/api/v0/vault/key", token, nil)
	expectStatus(t, "vault key", code, http.StatusOK)
	expectKey(t, "vault key", body, key)

	code, body = doObject(http.MethodPost, "/api/v0/vault/items", token, item(1))
	expectStatus(t, "create an item", code, http.StatusCreated)
	expectItem(t, "created item", body, item(1))
	itemPath := "/api/v0/vault/items/" + jsonNumber(body["id"])

	code, listed := do(http.MethodGet, "/api/v0/vault/items", token, nil)
	expectStatus(t, "list items", code, http.StatusOK)
	items, ok := listed.([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %v, want an array of the one item", listed)
	}
	listedItem, _ := items[0].(map[string]any)
	expectItem(t, "listed item", listedItem, item(1))

	code, body = doObject(http.MethodGet, itemPath, token, nil)
	expectStatus(t, "get an item", code, http.StatusOK)
	expectItem(t, "item", body, item(1))

	code, body = doObject(http.MethodPut, itemPath, token, item(2))
	expectStatus(t, "replace an item", code, http.StatusOK)
	expectItem(t, "replaced item", body, item(2))

	// Another account, the member, finds none of the admin's items.
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		code, body = doObject(method, itemPath, memberToken, item(3))
		expectStatus(t, "another account's "+method, code, http.StatusNotFound)
		expectString(t, "another account's "+method+" error", body, "error")
	}

	code, body = doObject(http.MethodGet, "/api/v0/vault/items/999999", token, nil)
	expectStatus(t, "get a missing item", code, http.StatusNotFound)
	expectString(t, "missing item error", body, "error")

	code, _ = doObject(http.MethodDelete, itemPath, token, nil)
	expectStatus(t, "delete an item", code, http.StatusNoContent)
	code, _ = doObject(http.MethodGet, itemPath, token, nil)
	expectStatus(t, "get a deleted item", code, http.StatusNotFound)

	preflight := httptest.NewRequest(http.MethodOptions, "/api/v0/vault/items", nil)
	preflight.Header.Set("Origin", "chrome-extension://abcdefghijklmnop")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	preflight.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, preflight)
	if w.Code != http.StatusNoContent && w.Code != http.StatusOK {
		t.Errorf("CORS preflight = %d, want 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want *", got)
	}
	if !strings.Contains(strings.ToLower(w.Header().Get("Access-Control-Allow-Headers")), "authorization") {
		t.Errorf("Access-Control-Allow-Headers = %q, want Authorization allowed", w.Header().Get("Access-Control-Allow-Headers"))
	}
	if w.Header().Get("Access-Control-Allow-Credentials") == "true" {
		t.Error("CORS must not allow credentials; the extension sends a bearer token, never cookies")
	}

	code, _ = doObject(http.MethodPost, "/api/v0/auth/logout", token, nil)
	expectStatus(t, "logout", code, http.StatusOK)
	code, _ = doObject(http.MethodGet, "/api/v0/vault/key", token, nil)
	expectStatus(t, "vault key after logout", code, http.StatusUnauthorized)
}

// b64 is the standard base64 of n bytes of fill.
func b64(fill byte, n int) string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{fill}, n))
}

func jsonNumber(id any) string {
	encoded, _ := json.Marshal(id)
	return string(encoded)
}

func expectStatus(t *testing.T, what string, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: status %d, want %d", what, got, want)
	}
}

func expectString(t *testing.T, what string, body map[string]any, key string) string {
	t.Helper()
	value, ok := body[key].(string)
	if !ok {
		t.Errorf("%s: %q is %T, want a string", what, key, body[key])
	}
	return value
}

func expectBool(t *testing.T, what string, body map[string]any, key string) {
	t.Helper()
	if _, ok := body[key].(bool); !ok {
		t.Errorf("%s: %q is %T, want a bool", what, key, body[key])
	}
}

func expectNumber(t *testing.T, what string, body map[string]any, key string) {
	t.Helper()
	if _, ok := body[key].(float64); !ok {
		t.Errorf("%s: %q is %T, want a number", what, key, body[key])
	}
}

// expectKey checks a served vault key carries the wraps and salts that were
// stored, as base64 strings, and kdfParams as the same object.
func expectKey(t *testing.T, what string, body, want map[string]any) {
	t.Helper()
	for _, field := range []string{"wrappedByPassword", "saltPw", "wrappedByPhrase", "saltRp"} {
		if got := expectString(t, what+"."+field, body, field); got != want[field] {
			t.Errorf("%s.%s = %q, want %q", what, field, got, want[field])
		}
	}
	params, ok := body["kdfParams"].(map[string]any)
	if !ok || params["alg"] != "argon2id13" {
		t.Errorf("%s.kdfParams = %v, want the object that was stored", what, body["kdfParams"])
	}
}

// expectItem checks a served vault item carries its id, the ciphertext and
// nonce that were stored as base64 strings, its formatVersion and timestamps.
func expectItem(t *testing.T, what string, body, want map[string]any) {
	t.Helper()
	expectNumber(t, what+".id", body, "id")
	for _, field := range []string{"ciphertext", "nonce"} {
		if got := expectString(t, what+"."+field, body, field); got != want[field] {
			t.Errorf("%s.%s = %q, want %q", what, field, got, want[field])
		}
	}
	expectNumber(t, what+".formatVersion", body, "formatVersion")
	expectString(t, what+".createdAt", body, "createdAt")
	expectString(t, what+".updatedAt", body, "updatedAt")
}
