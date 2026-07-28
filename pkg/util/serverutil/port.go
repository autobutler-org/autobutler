package serverutil

import (
	"os"
	"strconv"
	"sync/atomic"
)

// servingPort and servingTLS record the address the server actually bound, set
// once at startup by the server package. Callers that need to reach the server
// from inside the process (the tsnet remote-access proxy) must use these rather
// than guessing between ServerPort and ServerHttpsPort — guessing wrong means
// proxying plain HTTP at a TLS listener, or dialing a port nothing is on.
var (
	servingPort atomic.Int64
	servingTLS  atomic.Bool
)

// SetServingAddr records the port the server bound and whether it serves TLS.
func SetServingAddr(port int, tls bool) {
	servingPort.Store(int64(port))
	servingTLS.Store(tls)
}

// ServingPort returns the port the server actually bound. Falls back to the
// configured HTTPS/HTTP port when the server hasn't started yet.
func ServingPort() int {
	if p := servingPort.Load(); p != 0 {
		return int(p)
	}
	if servingTLS.Load() {
		return ServerHttpsPort()
	}
	return ServerPort()
}

// ServingTLS reports whether the server is serving HTTPS.
func ServingTLS() bool { return servingTLS.Load() }

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
