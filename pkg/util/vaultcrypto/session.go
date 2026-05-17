package vaultcrypto

import (
	"sync"
	"time"
)

// VaultSession holds the derived encryption key in memory while the vault is unlocked.
// The key is zeroed on lock or timeout.
type VaultSession struct {
	mu         sync.RWMutex
	key        []byte
	unlockedAt time.Time
	timeout    time.Duration
}

// NewVaultSession creates a new locked vault session.
func NewVaultSession() *VaultSession {
	return &VaultSession{}
}

// Unlock stores the derived key and starts the auto-lock timer.
func (s *VaultSession) Unlock(key []byte, timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Zero any previous key.
	if s.key != nil {
		ZeroKey(s.key)
	}

	s.key = make([]byte, len(key))
	copy(s.key, key)
	s.unlockedAt = time.Now()
	s.timeout = timeout
}

// Lock zeroes the key and marks the vault as locked.
func (s *VaultSession) Lock() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.key != nil {
		ZeroKey(s.key)
		s.key = nil
	}
}

// Key returns a copy of the encryption key if the vault is unlocked and has not timed out.
// Returns nil and false if locked or expired.
func (s *VaultSession) Key() ([]byte, bool) {
	s.mu.RLock()

	if s.key == nil {
		s.mu.RUnlock()
		return nil, false
	}

	if s.timeout > 0 && time.Since(s.unlockedAt) > s.timeout {
		s.mu.RUnlock()
		// Expired — lock and zero the key.
		s.Lock()
		return nil, false
	}

	keyCopy := make([]byte, len(s.key))
	copy(keyCopy, s.key)
	s.mu.RUnlock()
	return keyCopy, true
}

// Touch resets the auto-lock timer without changing the key.
func (s *VaultSession) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.key != nil {
		s.unlockedAt = time.Now()
	}
}

// IsLocked returns true if the vault is locked or the session has timed out.
func (s *VaultSession) IsLocked() bool {
	s.mu.RLock()

	if s.key == nil {
		s.mu.RUnlock()
		return true
	}

	if s.timeout > 0 && time.Since(s.unlockedAt) > s.timeout {
		s.mu.RUnlock()
		s.Lock()
		return true
	}

	s.mu.RUnlock()
	return false
}
