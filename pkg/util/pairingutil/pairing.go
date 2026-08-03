// Package pairingutil generates and validates short-lived pairing tokens used
// for the mobile QR onboarding flow (#1403). A pairing token encodes the
// butler's LAN address and port so a scanning phone can connect without typing.
package pairingutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	pairingTokenDuration = 10 * time.Minute
	signingKeySize       = 32 // bytes
)

// pairingSigner holds the signing key for pairing tokens. The key is generated
// once at startup; pairing tokens from previous runs are immediately invalid
// (which is fine — they're only meant to be used once in a short window).
var pairingSigner = &signer{}

type signer struct {
	mu  sync.RWMutex
	key []byte
}

func init() {
	if err := pairingSigner.rotate(); err != nil {
		panic(fmt.Sprintf("pairingutil: failed to generate signing key: %v", err))
	}
}

func (s *signer) rotate() error {
	key := make([]byte, signingKeySize)
	if _, err := rand.Read(key); err != nil {
		return err
	}
	s.mu.Lock()
	s.key = key
	s.mu.Unlock()
	return nil
}

func (s *signer) sign(claims jwt.Claims) (string, error) {
	s.mu.RLock()
	key := s.key
	s.mu.RUnlock()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(key)
}

func (s *signer) parse(tokenStr string, claims jwt.Claims) (*jwt.Token, error) {
	s.mu.RLock()
	key := s.key
	s.mu.RUnlock()
	return jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return key, nil
	})
}

// PairingClaims are the JWT claims embedded in a pairing token.
type PairingClaims struct {
	jwt.RegisteredClaims
	// ButlerAddr is the LAN address the scanning phone should connect to
	// (e.g. "192.168.1.42:443" or "192.168.1.42:8080").
	ButlerAddr string `json:"addr"`
	// Scheme is "https" or "http".
	Scheme string `json:"scheme"`
	// Nonce ensures each QR code is unique even if generated at the same second.
	Nonce string `json:"nonce"`
}

// IssuePairingToken generates a short-lived JWT encoding the butler's LAN
// address. The token is embedded in the QR code shown at /mobile.
func IssuePairingToken(butlerAddr, scheme string) (string, error) {
	nonce, err := randomHex(8)
	if err != nil {
		return "", fmt.Errorf("pairing token nonce: %w", err)
	}
	now := time.Now()
	claims := PairingClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "autobutler-pairing",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(pairingTokenDuration)),
		},
		ButlerAddr: butlerAddr,
		Scheme:     scheme,
		Nonce:      nonce,
	}
	return pairingSigner.sign(claims)
}

// ValidatePairingToken parses and validates a pairing token. Returns the
// decoded claims on success.
func ValidatePairingToken(tokenStr string) (*PairingClaims, error) {
	claims := &PairingClaims{}
	tok, err := pairingSigner.parse(tokenStr, claims)
	if err != nil || !tok.Valid {
		return nil, fmt.Errorf("invalid pairing token: %w", err)
	}
	return claims, nil
}

// LANAddress returns the best-guess LAN IP of this machine paired with the
// given port, in "ip:port" form. Falls back to "127.0.0.1:port" if detection
// fails.
func LANAddress(port int) string {
	// Dial a UDP address on the LAN (doesn't actually send data — we just
	// want the OS to pick the outbound interface's IP).
	conn, err := net.DialTimeout("udp", "192.168.1.1:53", time.Second)
	if err != nil {
		// Try a broader target.
		conn, err = net.DialTimeout("udp", "8.8.8.8:53", time.Second)
		if err != nil {
			return fmt.Sprintf("127.0.0.1:%d", port)
		}
	}
	defer conn.Close()
	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || localAddr == nil {
		return fmt.Sprintf("127.0.0.1:%d", port)
	}
	return fmt.Sprintf("%s:%d", localAddr.IP.String(), port)
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
