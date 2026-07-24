package serverutil

import (
	"os"
	"strconv"
)

// ServerPort returns the HTTP port the server listens on, read from the PORT
// environment variable. Defaults to 8080.
func ServerPort() int {
	p := os.Getenv("PORT")
	if p == "" {
		return 8080
	}
	n, err := strconv.Atoi(p)
	if err != nil {
		return 8080
	}
	return n
}

// ServerHttpsPort returns the HTTPS port the server listens on, read from the
// HTTPS_PORT environment variable. Defaults to 443.
//
// This is intentionally separate from ServerPort so that in-place binary
// upgrades on existing installations do not require a service-file edit: the
// old PORT=80 entry is left untouched and the new HTTPS_PORT=443 entry added
// only on fresh installs.
func ServerHttpsPort() int {
	p := os.Getenv("HTTPS_PORT")
	if p == "" {
		return 443
	}
	n, err := strconv.Atoi(p)
	if err != nil {
		return 443
	}
	return n
}
