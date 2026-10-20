package v0_vault_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	v0_vault "github.com/autobutler-org/quark/internal/server/api/v0/vault"
	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/gin-gonic/gin"
)

type harness struct {
	srv    *httptest.Server
	users  map[string]int64
	events <-chan eventbus.Event
}

// newHarness serves the vault router to admin, bob and carol, each named by
// the as query parameter the way requireAuth leaves a signed-in principal.
func newHarness(t *testing.T) harness {
	t.Helper()
	database := dbtest.NewDB(t)
	users := map[string]int64{}
	for _, name := range []string{"admin", "bob", "carol"} {
		hash, err := authutil.HashKey(dbtest.AuthKey(name + "-password"))
		if err != nil {
			t.Fatal(err)
		}
		user, err := database.Queries.CreateUser(context.Background(), db.CreateUserParams{Username: name, AuthKeyHash: hash, RecoveryPhraseHash: "r"})
		if err != nil {
			t.Fatal(err)
		}
		users[name] = user.ID
	}
	bus := eventbus.New()
	events, unsubscribe := bus.Subscribe("vault-integration-test")
	t.Cleanup(unsubscribe)

	deps := deputil.NewDependencies().WithEventBus(bus).WithDatabase(database)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		as := c.Query("as")
		c = ctxutil.With(c, "deps", deps)
		c = ctxutil.With(c, "principal", accessutil.Principal{UserID: users[as], IsAdmin: as == "admin"})
		c = ctxutil.With(c, "username", as)
		c.Next()
	})
	serverutil.RegisterRouterWithGroup(engine.Group("/api/v0"), v0_vault.NewRouter())
	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)
	return harness{srv: srv, users: users, events: events}
}

// expect sends body to path as the named account, fails unless the answer is
// want, and decodes the answer into out when out is not nil.
func (h harness) expect(t *testing.T, want int, method, path, as, body string, out any) {
	t.Helper()
	req, err := http.NewRequest(method, h.srv.URL+"/api/v0"+path+"?as="+as, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		t.Fatalf("%s %s as %s = %d %s, want %d", method, path, as, res.StatusCode, raw, want)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("%s %s: %v in %s", method, path, err, raw)
		}
	}
}

// heard drains the vault events published so far.
func (h harness) heard() []eventbus.VaultChanged {
	var out []eventbus.VaultChanged
	for {
		select {
		case evt := <-h.events:
			if data, ok := evt.Data.(eventbus.VaultChanged); ok && evt.Kind == eventbus.EventVaultChanged {
				out = append(out, data)
			}
		default:
			return out
		}
	}
}

func b64(fill byte, n int) string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{fill}, n))
}

// keyBody is a vault key wrapped under as's login password, with the auth key
// that re-confirms it.
func keyBody(as string, fill byte) string {
	return `{"password":"` + dbtest.AuthKey(as+"-password") + `","wrappedByPassword":"` + b64(fill, 72) + `","saltPw":"` + b64(fill, 16) +
		`","kdfParams":{"alg":"argon2id13","opsLimit":3,"memLimit":67108864}}`
}

func itemBody(fill byte) string {
	return `{"ciphertext":"` + b64(fill, 48) + `","nonce":"` + b64(fill, 24) + `","formatVersion":1}`
}

func TestVault_KeyAndItems(t *testing.T) {
	h := newHarness(t)
	bob := h.users["bob"]

	h.expect(t, http.StatusUnauthorized, http.MethodGet, "/vault/key", "stranger", "", nil)
	h.expect(t, http.StatusNotFound, http.MethodGet, "/vault/key", "bob", "", nil)
	h.expect(t, http.StatusConflict, http.MethodPost, "/vault/items", "bob", itemBody(1), nil)
	h.expect(t, http.StatusBadRequest, http.MethodPut, "/vault/key", "bob", `{"wrappedByPassword":"AAAA"}`, nil)
	h.expect(t, http.StatusRequestEntityTooLarge, http.MethodPut, "/vault/key", "bob", `{"pad":"`+strings.Repeat("x", vaultutil.MaxKeyRequestBytes)+`"}`, nil)
	// Only the login password may wrap the key: carol's auth key is not bob's.
	h.expect(t, http.StatusForbidden, http.MethodPut, "/vault/key", "bob", keyBody("carol", 1), nil)
	h.expect(t, http.StatusNotFound, http.MethodGet, "/vault/key", "bob", "", nil)
	if heard := h.heard(); len(heard) != 0 {
		t.Errorf("refused requests published %v", heard)
	}

	var key vaultutil.Key
	h.expect(t, http.StatusCreated, http.MethodPut, "/vault/key", "bob", keyBody("bob", 1), &key)
	if !bytes.Equal(key.SaltPw, bytes.Repeat([]byte{1}, 16)) || key.WrappedByPhrase != nil {
		t.Errorf("stored key = %+v", key)
	}
	h.expect(t, http.StatusConflict, http.MethodPut, "/vault/key", "bob", keyBody("bob", 2), nil)
	h.expect(t, http.StatusOK, http.MethodGet, "/vault/key", "bob", "", &key)
	if !bytes.Equal(key.WrappedByPassword, bytes.Repeat([]byte{1}, 72)) {
		t.Error("a second PUT /vault/key replaced the key")
	}

	var created vaultutil.Item
	h.expect(t, http.StatusCreated, http.MethodPost, "/vault/items", "bob", itemBody(1), &created)
	id := strconv.FormatInt(created.ID, 10)
	h.expect(t, http.StatusBadRequest, http.MethodPost, "/vault/items", "bob", `{"ciphertext":"","nonce":"AAAA","formatVersion":1}`, nil)
	h.expect(t, http.StatusRequestEntityTooLarge, http.MethodPost, "/vault/items", "bob",
		`{"ciphertext":"`+strings.Repeat("A", vaultutil.MaxItemRequestBytes)+`","nonce":"AAAA","formatVersion":1}`, nil)
	h.expect(t, http.StatusRequestEntityTooLarge, http.MethodPut, "/vault/items/"+id, "bob",
		`{"ciphertext":"`+strings.Repeat("A", vaultutil.MaxItemRequestBytes)+`","nonce":"AAAA","formatVersion":1}`, nil)

	var got vaultutil.Item
	h.expect(t, http.StatusOK, http.MethodGet, "/vault/items/"+id, "bob", "", &got)
	if !bytes.Equal(got.Ciphertext, bytes.Repeat([]byte{1}, 48)) || got.FormatVersion != 1 {
		t.Errorf("GET item = %+v", got)
	}
	h.expect(t, http.StatusOK, http.MethodPut, "/vault/items/"+id, "bob", itemBody(2), &got)
	if !bytes.Equal(got.Ciphertext, bytes.Repeat([]byte{2}, 48)) {
		t.Errorf("PUT item = %+v", got)
	}
	var listed []vaultutil.Item
	h.expect(t, http.StatusOK, http.MethodGet, "/vault/items", "bob", "", &listed)
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Errorf("list = %+v, want the one item", listed)
	}
	h.expect(t, http.StatusNotFound, http.MethodGet, "/vault/items/nope", "bob", "", nil)

	// Every change went to bob alone: the key, then the item twice.
	want := []eventbus.VaultChanged{{UserID: bob}, {ItemID: created.ID, UserID: bob}, {ItemID: created.ID, UserID: bob}}
	if heard := h.heard(); len(heard) != len(want) || heard[0] != want[0] || heard[1] != want[1] || heard[2] != want[2] {
		t.Errorf("events = %+v, want %+v", heard, want)
	}

	// Another member, and an admin, reach none of it and change nothing.
	for _, as := range []string{"carol", "admin"} {
		h.expect(t, http.StatusNotFound, http.MethodGet, "/vault/items/"+id, as, "", nil)
		h.expect(t, http.StatusNotFound, http.MethodPut, "/vault/items/"+id, as, itemBody(3), nil)
		h.expect(t, http.StatusNotFound, http.MethodDelete, "/vault/items/"+id, as, "", nil)
		h.expect(t, http.StatusNotFound, http.MethodGet, "/vault/key", as, "", nil)
		var theirs []vaultutil.Item
		h.expect(t, http.StatusOK, http.MethodGet, "/vault/items", as, "", &theirs)
		if theirs == nil || len(theirs) != 0 {
			t.Errorf("%s lists %v, want an empty array", as, theirs)
		}
	}
	h.expect(t, http.StatusOK, http.MethodGet, "/vault/items/"+id, "bob", "", &got)
	if !bytes.Equal(got.Ciphertext, bytes.Repeat([]byte{2}, 48)) {
		t.Error("another account changed bob's item")
	}
	if heard := h.heard(); len(heard) != 0 {
		t.Errorf("refused requests from other accounts published %v", heard)
	}

	h.expect(t, http.StatusNoContent, http.MethodDelete, "/vault/items/"+id, "bob", "", nil)
	h.expect(t, http.StatusNotFound, http.MethodDelete, "/vault/items/"+id, "bob", "", nil)
	if heard := h.heard(); len(heard) != 1 || heard[0] != (eventbus.VaultChanged{ItemID: created.ID, UserID: bob}) {
		t.Errorf("delete events = %+v", heard)
	}
}

// TestVault_DeleteVault wipes one account's items and key, and lets it start
// over.
func TestVault_DeleteVault(t *testing.T) {
	h := newHarness(t)
	for _, as := range []string{"bob", "carol"} {
		h.expect(t, http.StatusCreated, http.MethodPut, "/vault/key", as, keyBody(as, 1), nil)
		h.expect(t, http.StatusCreated, http.MethodPost, "/vault/items", as, itemBody(1), nil)
	}
	h.heard()

	h.expect(t, http.StatusNoContent, http.MethodDelete, "/vault", "bob", "", nil)
	if heard := h.heard(); len(heard) != 1 || heard[0] != (eventbus.VaultChanged{UserID: h.users["bob"]}) {
		t.Errorf("events = %+v, want one vault-level change for bob", heard)
	}
	h.expect(t, http.StatusNotFound, http.MethodGet, "/vault/key", "bob", "", nil)
	var listed []vaultutil.Item
	h.expect(t, http.StatusOK, http.MethodGet, "/vault/items", "bob", "", &listed)
	if len(listed) != 0 {
		t.Errorf("bob still has %d items", len(listed))
	}
	h.expect(t, http.StatusOK, http.MethodGet, "/vault/items", "carol", "", &listed)
	if len(listed) != 1 {
		t.Errorf("carol has %d items after bob's wipe, want 1", len(listed))
	}
	h.expect(t, http.StatusOK, http.MethodGet, "/vault/key", "carol", "", nil)

	h.expect(t, http.StatusNoContent, http.MethodDelete, "/vault", "bob", "", nil)
	h.expect(t, http.StatusCreated, http.MethodPut, "/vault/key", "bob", keyBody("bob", 2), nil)
}
