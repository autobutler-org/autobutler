package frame_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/autobutler-org/quark/internal/db/dbtest"
	v0_auth "github.com/autobutler-org/quark/internal/server/api/v0/auth"
	"github.com/autobutler-org/quark/internal/server/frame"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/settingsutil"
	"github.com/autobutler-org/quark/pkg/vfs"
	"github.com/gin-gonic/gin"
)

func post(e *gin.Engine, target string, body any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// The real auth routes on a real database, with every key derived by the page's own JavaScript: the account is
// created with a key the page derived, and the page's key for the same password signs in. A different password's key
// is refused. This is what the sign-in on the frame depends on, with nothing in between faked.
func TestServerAcceptsTheKeyThePageDerives(t *testing.T) {
	settingsutil.ResetForTesting(filepath.Join(t.TempDir(), "settings.json"))
	t.Cleanup(func() { settingsutil.ResetForTesting("") })
	registry := vfs.NewRegistry()
	if err := registry.Register(vfs.Namespace{ID: vfs.FilesNamespace("")}, vfs.NewMemVFS("files")); err != nil {
		t.Fatal(err)
	}
	deps := deputil.NewDependencies().WithDatabase(dbtest.NewDB(t)).WithVFSRegistry(registry)
	gin.SetMode(gin.TestMode)
	e := gin.New()
	e.Use(func(c *gin.Context) {
		c = ctxutil.With(c, "deps", deps)
		c.Next()
	})
	serverutil.RegisterRouterWithGroup(e.Group("/api/v0"), v0_auth.NewRouter())
	frame.Register(e)

	// The salt a username gets is the same before and after the account exists.
	salt := func() string {
		rec := get(e, "/api/v0/auth/salt?username=owner")
		var body struct {
			Salt string `json:"salt"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Salt == "" {
			t.Fatalf("salt = %d %s", rec.Code, rec.Body.String())
		}
		return body.Salt
	}
	keys := pageDerive(t, salt(), "a long enough password", "some other password")

	if rec := post(e, "/api/v0/auth/setup", map[string]string{"username": "owner", "authKey": keys[0]}); rec.Code != http.StatusOK {
		t.Fatalf("setup with the page's key = %d: %s", rec.Code, rec.Body.String())
	}
	rec := post(e, "/api/v0/auth/login", map[string]string{"username": "owner", "authKey": keys[0]})
	if rec.Code != http.StatusOK {
		t.Fatalf("login with the page's key = %d: %s", rec.Code, rec.Body.String())
	}
	var login struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &login); err != nil || login.Token == "" {
		t.Errorf("login gave no token: %s", rec.Body.String())
	}
	if cookies := rec.Result().Cookies(); len(cookies) == 0 || cookies[0].Name != "session" {
		t.Errorf("login set no session cookie, which the page depends on: %v", cookies)
	}
	if rec := post(e, "/api/v0/auth/login", map[string]string{"username": "owner", "authKey": keys[1]}); rec.Code != http.StatusUnauthorized {
		t.Errorf("login with another password's key = %d, want 401", rec.Code)
	}
}
