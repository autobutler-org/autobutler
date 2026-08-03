package pairingutil

import (
	"strings"
	"testing"
)

func TestIssuePairingToken_RoundTrip(t *testing.T) {
	addr := "192.168.1.42:443"
	scheme := "https"

	token, err := IssuePairingToken(addr, scheme)
	if err != nil {
		t.Fatalf("IssuePairingToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := ValidatePairingToken(token)
	if err != nil {
		t.Fatalf("ValidatePairingToken: %v", err)
	}
	if claims.ButlerAddr != addr {
		t.Errorf("addr: got %q, want %q", claims.ButlerAddr, addr)
	}
	if claims.Scheme != scheme {
		t.Errorf("scheme: got %q, want %q", claims.Scheme, scheme)
	}
	if claims.Nonce == "" {
		t.Error("expected non-empty nonce")
	}
}

func TestIssuePairingToken_Uniqueness(t *testing.T) {
	tok1, _ := IssuePairingToken("192.168.1.1:443", "https")
	tok2, _ := IssuePairingToken("192.168.1.1:443", "https")
	if tok1 == tok2 {
		t.Error("expected distinct tokens for sequential calls (nonce should differ)")
	}
}

func TestValidatePairingToken_Invalid(t *testing.T) {
	_, err := ValidatePairingToken("not.a.token")
	if err == nil {
		t.Error("expected error for invalid token, got nil")
	}
}

func TestValidatePairingToken_Tampered(t *testing.T) {
	token, _ := IssuePairingToken("192.168.1.1:443", "https")
	// Flip the last character
	tampered := token[:len(token)-1] + "X"
	if strings.HasSuffix(token, "X") {
		tampered = token[:len(token)-1] + "Y"
	}
	_, err := ValidatePairingToken(tampered)
	if err == nil {
		t.Error("expected error for tampered token, got nil")
	}
}

func TestLANAddress_Returns(t *testing.T) {
	addr := LANAddress(443)
	if addr == "" {
		t.Error("expected non-empty LAN address")
	}
	// Should be in ip:port form
	if !strings.Contains(addr, ":") {
		t.Errorf("expected 'ip:port' form, got %q", addr)
	}
}
