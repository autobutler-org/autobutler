package remoteutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"
)

func selfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}
}

func TestTLSNegotiatesPQKeyExchange(t *testing.T) {
	cert := selfSignedCert(t)

	serverCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", serverCfg)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	done := make(chan tls.ConnectionState, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tlsConn := conn.(*tls.Conn)
		if err := tlsConn.Handshake(); err != nil {
			return
		}
		done <- tlsConn.ConnectionState()
	}()

	certPool := x509.NewCertPool()
	certPool.AddCert(mustParseCert(t, cert.Certificate[0]))

	clientCfg := &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS13,
	}

	conn, err := tls.Dial("tcp", ln.Addr().String(), clientCfg)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	select {
	case state := <-done:
		if !isPostQuantumCurve(state.CurveID) {
			t.Errorf("expected post-quantum key exchange, got %s (ID %d)", state.CurveID, state.CurveID)
		}
		t.Logf("negotiated: %s (CurveID %d)", state.CurveID, state.CurveID)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for server handshake")
	}
}

func TestTLSExplicitPQPreferences(t *testing.T) {
	cert := selfSignedCert(t)

	serverCfg := &tls.Config{
		Certificates:     []tls.Certificate{cert},
		MinVersion:       tls.VersionTLS13,
		CurvePreferences: PQCurvePreferences(),
	}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", serverCfg)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	done := make(chan tls.ConnectionState, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tlsConn := conn.(*tls.Conn)
		if err := tlsConn.Handshake(); err != nil {
			return
		}
		done <- tlsConn.ConnectionState()
	}()

	certPool := x509.NewCertPool()
	certPool.AddCert(mustParseCert(t, cert.Certificate[0]))

	clientCfg := &tls.Config{
		RootCAs:          certPool,
		MinVersion:       tls.VersionTLS13,
		CurvePreferences: PQCurvePreferences(),
	}

	conn, err := tls.Dial("tcp", ln.Addr().String(), clientCfg)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	select {
	case state := <-done:
		if state.CurveID != tls.X25519MLKEM768 {
			t.Errorf("expected X25519MLKEM768 (4588), got %s (ID %d)", state.CurveID, state.CurveID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for server handshake")
	}
}

func TestTLSFallsBackToClassical(t *testing.T) {
	cert := selfSignedCert(t)

	serverCfg := &tls.Config{
		Certificates:     []tls.Certificate{cert},
		MinVersion:       tls.VersionTLS13,
		CurvePreferences: []tls.CurveID{tls.X25519},
	}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", serverCfg)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	done := make(chan tls.ConnectionState, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		tlsConn := conn.(*tls.Conn)
		if err := tlsConn.Handshake(); err != nil {
			return
		}
		done <- tlsConn.ConnectionState()
	}()

	certPool := x509.NewCertPool()
	certPool.AddCert(mustParseCert(t, cert.Certificate[0]))

	clientCfg := &tls.Config{
		RootCAs:          certPool,
		MinVersion:       tls.VersionTLS13,
		CurvePreferences: PQCurvePreferences(),
	}

	conn, err := tls.Dial("tcp", ln.Addr().String(), clientCfg)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	select {
	case state := <-done:
		if state.CurveID != tls.X25519 {
			t.Errorf("expected X25519 fallback when server only offers classical, got %s", state.CurveID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for server handshake")
	}
}

func TestPQCurvePreferences(t *testing.T) {
	prefs := PQCurvePreferences()
	if len(prefs) < 2 {
		t.Fatalf("expected at least 2 curve preferences, got %d", len(prefs))
	}
	if prefs[0] != tls.X25519MLKEM768 {
		t.Errorf("first preference should be X25519MLKEM768, got %s", prefs[0])
	}
	hasFallback := false
	for _, c := range prefs {
		if c == tls.X25519 {
			hasFallback = true
		}
	}
	if !hasFallback {
		t.Error("preferences must include X25519 as classical fallback")
	}
}

func TestIsPostQuantumCurve(t *testing.T) {
	tests := []struct {
		id   tls.CurveID
		want bool
	}{
		{tls.X25519MLKEM768, true},
		{tls.SecP256r1MLKEM768, true},
		{tls.SecP384r1MLKEM1024, true},
		{tls.X25519, false},
		{tls.CurveP256, false},
		{tls.CurveP384, false},
	}
	for _, tt := range tests {
		if got := isPostQuantumCurve(tt.id); got != tt.want {
			t.Errorf("isPostQuantumCurve(%s) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func mustParseCert(t *testing.T, der []byte) *x509.Certificate {
	t.Helper()
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}
	return cert
}
