// Package tlsutil provides helpers for provisioning self-signed TLS certificates
// used by the Autobutler server when running in production (HTTPS) mode.
package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
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
// valid for 365 days with SANs for localhost, 127.0.0.1, ::1, and any local network IPs.
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

// needsRegen returns true when the cert file is absent, unreadable, or expires
// within the renewalWindow.
func needsRegen(certFile string) bool {
	data, err := os.ReadFile(certFile)
	if err != nil {
		return true // absent or unreadable
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return true
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return true
	}
	return time.Until(cert.NotAfter) < renewalWindow
}

// generate creates a new ECDSA P-256 self-signed certificate with SANs covering
// localhost, loopback addresses, and all non-loopback interface IPs.
// P-256 is the recommended key type for TLS 1.3 certificates — compact, fast,
// and well-supported. Go 1.22+ automatically negotiates X25519MLKEM768 hybrid
// PQC key exchange in TLS 1.3, so session keys are post-quantum hybrid without
// any extra configuration.
func generate(certFile, keyFile string) error {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate ECDSA key: %w", err)
	}

	// Collect SANs.
	dnsNames := []string{"localhost"}
	ipAddrs := []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("::1"),
	}
	ifaceIPs, _ := localInterfaceIPs()
	ipAddrs = append(ipAddrs, ifaceIPs...)

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generate serial: %w", err)
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"Autobutler Self-Signed"},
			CommonName:   "autobutler-local",
		},
		NotBefore:             now.Add(-1 * time.Minute), // slight back-date for clock skew
		NotAfter:              now.Add(certValidity),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddrs,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &privKey.PublicKey, privKey)
	if err != nil {
		return fmt.Errorf("create certificate: %w", err)
	}

	// Write cert file.
	cf, err := os.OpenFile(certFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open cert file: %w", err)
	}
	defer cf.Close()
	if err := pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("encode cert PEM: %w", err)
	}

	// Write key file — ECDSA keys are marshalled as SEC 1 / EC PRIVATE KEY.
	keyDER, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("marshal EC key: %w", err)
	}
	kf, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open key file: %w", err)
	}
	defer kf.Close()
	if err := pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}); err != nil {
		return fmt.Errorf("encode key PEM: %w", err)
	}

	return nil
}

// localInterfaceIPs returns all non-loopback unicast IP addresses from the
// host's network interfaces, used to populate certificate SANs so that LAN
// clients can connect without certificate hostname errors.
func localInterfaceIPs() ([]net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}
	var ips []net.IP
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsLoopback() {
			continue
		}
		ips = append(ips, ip)
	}
	return ips, nil
}
