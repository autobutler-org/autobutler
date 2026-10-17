package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/internal/server/middleware"
	"github.com/autobutler-org/quark/pkg/util/featureflagutil"
	"github.com/autobutler-org/quark/pkg/util/settingsutil"
	"github.com/gin-gonic/gin"
)

// TestRequireFeatureEnabled_RetiredKeyStaysOff checks a key the registry no
// longer declares, slides after it left beta (#2869), cannot be turned on
// and its gate stays a 404.
func TestRequireFeatureEnabled_RetiredKeyStaysOff(t *testing.T) {
	settingsutil.ResetForTesting(filepath.Join(t.TempDir(), "settings.json"))
	const key = "slides"
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Group("", middleware.RequireFeatureEnabled(key)).GET("/slides", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	get := func() (int, string) {
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/slides", nil))
		return w.Code, w.Body.String()
	}

	if code, body := get(); code != http.StatusNotFound || !strings.Contains(body, "turned off") {
		t.Fatalf("GET = %d %s, want 404 while slides is unknown", code, body)
	}
	if _, err := featureflagutil.SetFlag(featureflagutil.SetFlagParams{Key: key, Enabled: true}); !errors.Is(err, featureflagutil.ErrUnknownFlag) {
		t.Fatalf("SetFlag = %v; want ErrUnknownFlag", err)
	}
	if code, body := get(); code != http.StatusNotFound || !strings.Contains(body, "turned off") {
		t.Fatalf("GET after SetFlag = %d %s, want 404", code, body)
	}
}
