// Package ratelimitutil provides a per-IP token-bucket rate limiter for
// protecting sensitive endpoints (e.g. login, setup, recover) from brute-force
// and credential-stuffing attacks.
package ratelimitutil

import (
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	// DefaultRate is the number of requests allowed per second per IP.
	DefaultRate rate.Limit = 5
	// DefaultBurst is the maximum burst size for a single IP.
	DefaultBurst = 10
	// cleanupInterval is how often expired entries are removed from the map.
	cleanupInterval = 5 * time.Minute
	// ttl is how long an idle IP entry is kept before being removed.
	ttl = 15 * time.Minute
)

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Limiter holds per-IP rate limiters.
type Limiter struct {
	mu      sync.Mutex
	entries map[string]*entry
	r       rate.Limit
	b       int
}

// New returns a Limiter with the default rate and burst.
func New() *Limiter {
	return NewWithRate(DefaultRate, DefaultBurst)
}

// NewWithRate returns a Limiter with custom rate (per second) and burst.
func NewWithRate(r rate.Limit, b int) *Limiter {
	l := &Limiter{
		entries: make(map[string]*entry),
		r:       r,
		b:       b,
	}
	go l.cleanup()
	return l
}

// Allow returns true if the request from ip is within the rate limit.
func (l *Limiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	e, ok := l.entries[ip]
	if !ok {
		e = &entry{limiter: rate.NewLimiter(l.r, l.b)}
		l.entries[ip] = e
	}
	e.lastSeen = time.Now()
	return e.limiter.Allow()
}

// AllowN returns true if n tokens are available for ip.
func (l *Limiter) AllowN(ip string, n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	e, ok := l.entries[ip]
	if !ok {
		e = &entry{limiter: rate.NewLimiter(l.r, l.b)}
		l.entries[ip] = e
	}
	e.lastSeen = time.Now()
	return e.limiter.AllowN(time.Now(), n)
}

// cleanup periodically removes entries that haven't been seen for ttl duration.
func (l *Limiter) cleanup() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		for ip, e := range l.entries {
			if time.Since(e.lastSeen) > ttl {
				delete(l.entries, ip)
			}
		}
		l.mu.Unlock()
	}
}

// ExtractIP returns the client IP from a raw address string, stripping the port.
// Falls back to the full address if parsing fails.
func ExtractIP(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}
