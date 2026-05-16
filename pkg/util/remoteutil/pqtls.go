package remoteutil

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"
)

// PQCurvePreferences returns the ordered key exchange preferences for
// AutoButler TLS connections. X25519MLKEM768 (hybrid post-quantum) is
// preferred, with classical X25519 as fallback for incompatible servers.
func PQCurvePreferences() []tls.CurveID {
	return []tls.CurveID{
		tls.X25519MLKEM768,
		tls.X25519,
	}
}

func isPostQuantumCurve(id tls.CurveID) bool {
	switch id {
	case tls.X25519MLKEM768, tls.SecP256r1MLKEM768, tls.SecP384r1MLKEM1024:
		return true
	default:
		return false
	}
}

// AssertPQKeyExchange checks whether a TLS connection state used a
// post-quantum key exchange. Returns nil if PQ was negotiated, or an error
// describing what was used instead.
func AssertPQKeyExchange(state tls.ConnectionState) error {
	if isPostQuantumCurve(state.CurveID) {
		return nil
	}
	return fmt.Errorf("pqtls: expected post-quantum key exchange, got %s (CurveID %d)", state.CurveID, state.CurveID)
}

// LogPQStatus logs whether a TLS connection negotiated post-quantum key
// exchange. Intended for use after establishing control plane connections.
func LogPQStatus(state tls.ConnectionState, peer string) {
	if isPostQuantumCurve(state.CurveID) {
		log.Printf("[pqtls] %s: post-quantum key exchange active (%s)", peer, state.CurveID)
	} else {
		log.Printf("[pqtls] WARNING: %s: classical key exchange only (%s) — vulnerable to HNDL", peer, state.CurveID)
	}
}

// verifyControlPlanePQ dials the Headscale control URL over TLS and logs
// whether post-quantum key exchange was negotiated. Runs asynchronously
// after tsnet startup — failure is non-fatal (logged only).
func verifyControlPlanePQ(controlURL string) {
	u, err := url.Parse(controlURL)
	if err != nil || u.Scheme != "https" {
		return
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}
	addr := net.JoinHostPort(host, port)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialer := tls.Dialer{
		Config: &tls.Config{
			ServerName:       host,
			CurvePreferences: PQCurvePreferences(),
			MinVersion:       tls.VersionTLS13,
		},
	}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		log.Printf("[pqtls] could not verify control plane: %v", err)
		return
	}
	defer conn.Close()

	state := conn.(*tls.Conn).ConnectionState()
	LogPQStatus(state, "headscale")
}
