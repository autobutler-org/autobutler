// Package tlsutil provides helpers for provisioning self-signed TLS certificates
// used by the Quark server when running in production (HTTPS) mode.
package tlsutil

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	renewalWindow = 30 * 24 * time.Hour // Regenerate if cert expires within 30 days.
	certValidity  = 365 * 24 * time.Hour
)

// EnsureSelfSignedCert checks if certFile/keyFile exist and are valid (not expiring soon).
// If absent or expiring within 30 days, it generates a new ECDSA P-256 self-signed cert
// valid for 365 days with SANs for localhost, the machine's own hostname and its
// .local mDNS name, 127.0.0.1, ::1, and any local network IPs.
// Cert and key are stored at dataDir/certs/server.crt and server.key.
// Returns the paths to the cert and key files.
func EnsureSelfSignedCert(dataDir string) (certFile, keyFile string, err error) {
	certsDir := filepath.Join(dataDir, "certs")
	certFile = filepath.Join(certsDir, "server.crt")
	keyFile = filepath.Join(certsDir, "server.key")

	if needsRegen(certFile) {
		log.Printf("[tlsutil] generating new self-signed TLS certificate in %s", certsDir)
		if err = os.MkdirAll(certsDir, 0o700); err != nil {
			return "", "", fmt.Errorf("tlsutil: create certs dir: %w", err)
		}
		if err = generate(certFile, keyFile); err != nil {
			return "", "", fmt.Errorf("tlsutil: generate cert: %w", err)
		}
		log.Printf("[tlsutil] self-signed cert written to %s", certFile)
	} else {
		log.Printf("[tlsutil] reusing existing TLS certificate at %s", certFile)
	}

	return certFile, keyFile, nil
}
