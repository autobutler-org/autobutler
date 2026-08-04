package v0_auth

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func ginCtxWithRequest(r *http.Request) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = r
	return c
}

func TestSecureCookie_PlainHTTP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No TLS, no forwarded headers → insecure mode
	c := ginCtxWithRequest(req)
	if secureCookie(c) {
		t.Error("plain HTTP request should return false")
	}
}

func TestSecureCookie_TLS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{} // non-nil TLS signals HTTPS
	c := ginCtxWithRequest(req)
	if !secureCookie(c) {
		t.Error("request with TLS set should return true")
	}
}

func TestSecureCookie_ForwardedProto(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	c := ginCtxWithRequest(req)
	if !secureCookie(c) {
		t.Error("X-Forwarded-Proto: https should return true")
	}
}

func TestSecureCookie_ForwardedProtoHTTP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "http")
	c := ginCtxWithRequest(req)
	if secureCookie(c) {
		t.Error("X-Forwarded-Proto: http should return false")
	}
}

func TestSecureCookie_ForwardedSsl(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Ssl", "on")
	c := ginCtxWithRequest(req)
	if !secureCookie(c) {
		t.Error("X-Forwarded-Ssl: on should return true")
	}
}

func TestSecureCookie_ForwardedSslOff(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Ssl", "off")
	c := ginCtxWithRequest(req)
	if secureCookie(c) {
		t.Error("X-Forwarded-Ssl: off should return false")
	}
}
