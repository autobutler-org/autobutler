package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobutler-org/quark/internal/server/middleware"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/gin-gonic/gin"
)

// TestRequestAccountPath_ExemptAndRateLimited checks someone with no session
// can reach the account request path, and that the path shares the per-IP
// auth limiter with login (#1908).
//
// The graph has no database on purpose. requireAuth answers 503 for any path
// it would authenticate when there is none, so reaching the handler proves the
// path is exempt; and with no database, trackDevice writes nothing from its
// goroutine into the test's temporary directory while it is being removed.
func TestRequestAccountPath_ExemptAndRateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	middleware.Use(engine, deputil.NewDependencies())
	const path = "/api/v0/auth/request-account"
	engine.POST(path, func(c *gin.Context) { c.Status(http.StatusCreated) })
	engine.POST("/api/v0/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	if w := doMiddlewareReq(engine, httptest.NewRequest(http.MethodPost, "/api/v0/protected", nil)); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("control: a non-exempt path = %d, want 503 with no database", w.Code)
	}
	if w := doMiddlewareReq(engine, httptest.NewRequest(http.MethodPost, path, nil)); w.Code != http.StatusCreated {
		t.Fatalf("anonymous request = %d, want it to reach the handler", w.Code)
	}
	for range 50 {
		if doMiddlewareReq(engine, httptest.NewRequest(http.MethodPost, path, nil)).Code == http.StatusTooManyRequests {
			return
		}
	}
	t.Error("50 requests from one IP never hit 429")
}

// TestVaultKeyPut_SharesTheAuthLimit checks that setting a vault up, which
// re-confirms the login password, shares the per-IP sign-in limit, while
// reading the key back does not (#3461).
func TestVaultKeyPut_SharesTheAuthLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	middleware.Use(engine, deputil.NewDependencies())
	const path = "/api/v0/vault/key"
	for range 50 {
		if doMiddlewareReq(engine, httptest.NewRequest(http.MethodGet, path, nil)).Code == http.StatusTooManyRequests {
			t.Fatal("GET of the vault key hit the sign-in limit")
		}
	}
	for range 50 {
		if doMiddlewareReq(engine, httptest.NewRequest(http.MethodPut, path, nil)).Code == http.StatusTooManyRequests {
			return
		}
	}
	t.Error("50 PUTs from one IP never hit 429")
}

// TestVaultDelete_SharesTheAuthLimit checks that deleting a vault, which
// re-confirms the login password, shares the per-IP sign-in limit (#3615).
func TestVaultDelete_SharesTheAuthLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	middleware.Use(engine, deputil.NewDependencies())
	const path = "/api/v0/vault"
	for range 50 {
		if doMiddlewareReq(engine, httptest.NewRequest(http.MethodGet, path, nil)).Code == http.StatusTooManyRequests {
			t.Fatal("GET of the vault hit the sign-in limit")
		}
	}
	for range 50 {
		if doMiddlewareReq(engine, httptest.NewRequest(http.MethodDelete, path, nil)).Code == http.StatusTooManyRequests {
			return
		}
	}
	t.Error("50 DELETEs from one IP never hit 429")
}
