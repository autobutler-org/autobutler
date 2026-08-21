package provisionutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestProvisioningURL_DefaultWhenEnvUnset verifies the built-in default is
// returned when QUARK_PROVISIONING_URL is not set.
func TestProvisioningURL_DefaultWhenEnvUnset(t *testing.T) {
	t.Setenv("QUARK_PROVISIONING_URL", "")
	if got := ProvisioningURL(); got != defaultProvisioningURL {
		t.Errorf("ProvisioningURL() = %q; want %q", got, defaultProvisioningURL)
	}
}

// TestProvisioningURL_OverriddenByEnv verifies the env var overrides the default.
func TestProvisioningURL_OverriddenByEnv(t *testing.T) {
	custom := "https://provision.example.internal"
	t.Setenv("QUARK_PROVISIONING_URL", custom)
	if got := ProvisioningURL(); got != custom {
		t.Errorf("ProvisioningURL() = %q; want %q", got, custom)
	}
}

// TestProvisioningSecret_EmptyWhenUnset verifies empty string returned when
// the env var is absent.
func TestProvisioningSecret_EmptyWhenUnset(t *testing.T) {
	t.Setenv("QUARK_PROVISIONING_SECRET", "")
	if got := ProvisioningSecret(); got != "" {
		t.Errorf("ProvisioningSecret() = %q; want empty string", got)
	}
}

// TestProvisioningSecret_FromEnv verifies the secret is read from env.
func TestProvisioningSecret_FromEnv(t *testing.T) {
	t.Setenv("QUARK_PROVISIONING_SECRET", "super-secret-token")
	if got := ProvisioningSecret(); got != "super-secret-token" {
		t.Errorf("ProvisioningSecret() = %q; want 'super-secret-token'", got)
	}
}

// TestProvisionAuthKey_ReturnsKeyOnSuccess verifies that ProvisionAuthKey
// parses a successful 200 response and returns the auth key.
func TestProvisionAuthKey_ReturnsKeyOnSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/provision" {
			t.Errorf("expected /provision, got %s", r.URL.Path)
		}
		if r.Header.Get("X-Provisioning-Secret") != "test-secret" {
			t.Errorf("missing or wrong secret header: %q", r.Header.Get("X-Provisioning-Secret"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(provisionResponse{AuthKey: "tskey-auth-12345"})
	}))
	defer ts.Close()

	t.Setenv("QUARK_PROVISIONING_URL", ts.URL)
	t.Setenv("QUARK_PROVISIONING_SECRET", "test-secret")

	key, err := ProvisionAuthKey("device-abc")
	if err != nil {
		t.Fatalf("ProvisionAuthKey: %v", err)
	}
	if key != "tskey-auth-12345" {
		t.Errorf("ProvisionAuthKey() = %q; want 'tskey-auth-12345'", key)
	}
}

// TestProvisionAuthKey_ErrorOnNonOKStatus verifies that a non-200 response
// from the provisioning service is surfaced as an error.
func TestProvisionAuthKey_ErrorOnNonOKStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer ts.Close()

	t.Setenv("QUARK_PROVISIONING_URL", ts.URL)
	t.Setenv("QUARK_PROVISIONING_SECRET", "bad-secret")

	_, err := ProvisionAuthKey("device-xyz")
	if err == nil {
		t.Fatal("expected error on 401 response, got nil")
	}
}

// TestProvisionAuthKey_ErrorWhenSecretMissing verifies that missing
// QUARK_PROVISIONING_SECRET is caught before any network call.
func TestProvisionAuthKey_ErrorWhenSecretMissing(t *testing.T) {
	t.Setenv("QUARK_PROVISIONING_SECRET", "")
	t.Setenv("QUARK_PROVISIONING_URL", "https://127.0.0.1:0") // unreachable — should not be called

	_, err := ProvisionAuthKey("device-xyz")
	if err == nil {
		t.Fatal("expected error when secret is missing, got nil")
	}
}

// TestProvisionAuthKey_ErrorOnEmptyAuthKey verifies that a 200 response with
// an empty auth_key field is treated as an error.
func TestProvisionAuthKey_ErrorOnEmptyAuthKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(provisionResponse{AuthKey: ""})
	}))
	defer ts.Close()

	t.Setenv("QUARK_PROVISIONING_URL", ts.URL)
	t.Setenv("QUARK_PROVISIONING_SECRET", "some-secret")

	_, err := ProvisionAuthKey("device-abc")
	if err == nil {
		t.Fatal("expected error on empty auth key, got nil")
	}
}

func TestProvisioningURL_DefaultIsHTTPS(t *testing.T) {
	t.Setenv("QUARK_PROVISIONING_URL", "")
	os.Unsetenv("QUARK_PROVISIONING_URL")
	url := ProvisioningURL()
	if !strings.HasPrefix(url, "https://") {
		t.Errorf("default provisioning URL should be HTTPS, got %q", url)
	}
}
