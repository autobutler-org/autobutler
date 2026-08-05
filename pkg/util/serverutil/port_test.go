package serverutil

import (
	"testing"
)

func TestServerPort_Default(t *testing.T) {
	t.Setenv("PORT", "")
	if got := ServerPort(); got != 8080 {
		t.Errorf("ServerPort() = %d; want 8080 (default)", got)
	}
}

func TestServerPort_FromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	if got := ServerPort(); got != 9090 {
		t.Errorf("ServerPort() = %d; want 9090", got)
	}
}

func TestServerPort_InvalidEnv(t *testing.T) {
	t.Setenv("PORT", "not-a-number")
	if got := ServerPort(); got != 8080 {
		t.Errorf("ServerPort() = %d; want 8080 (default on parse error)", got)
	}
}

func TestServerHttpsPort_Default(t *testing.T) {
	t.Setenv("HTTPS_PORT", "")
	if got := ServerHttpsPort(); got != 443 {
		t.Errorf("ServerHttpsPort() = %d; want 443 (default)", got)
	}
}

func TestServerHttpsPort_FromEnv(t *testing.T) {
	t.Setenv("HTTPS_PORT", "8443")
	if got := ServerHttpsPort(); got != 8443 {
		t.Errorf("ServerHttpsPort() = %d; want 8443", got)
	}
}

func TestServerHttpsPort_InvalidEnv(t *testing.T) {
	t.Setenv("HTTPS_PORT", "bad")
	if got := ServerHttpsPort(); got != 443 {
		t.Errorf("ServerHttpsPort() = %d; want 443 (default on parse error)", got)
	}
}

func TestSetServingAddr_AndRead(t *testing.T) {
	// Reset after test to avoid polluting other tests in the same process.
	t.Cleanup(func() { SetServingAddr(0, false) })

	SetServingAddr(7443, true)

	if got := ServingPort(); got != 7443 {
		t.Errorf("ServingPort() = %d; want 7443", got)
	}
	if !ServingTLS() {
		t.Error("ServingTLS() = false; want true")
	}
}

func TestServingPort_FallbackToHttpsWhenTLS(t *testing.T) {
	t.Cleanup(func() { SetServingAddr(0, false) })
	t.Setenv("HTTPS_PORT", "443")

	// Simulate server not yet started (port=0) but TLS mode set.
	SetServingAddr(0, true)

	if got := ServingPort(); got != 443 {
		t.Errorf("ServingPort() = %d; want 443 (HTTPS fallback)", got)
	}
}

func TestServingPort_FallbackToHttpWhenNoTLS(t *testing.T) {
	t.Cleanup(func() { SetServingAddr(0, false) })
	t.Setenv("PORT", "8080")

	SetServingAddr(0, false)

	if got := ServingPort(); got != 8080 {
		t.Errorf("ServingPort() = %d; want 8080 (HTTP fallback)", got)
	}
}
