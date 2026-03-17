package authutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost       = 12
	sessionTokenSize = 32 // bytes → 64 hex chars
	recoveryWords    = 6
)

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword verifies a plaintext password against a bcrypt hash.
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateSessionToken returns a cryptographically random hex session token.
func GenerateSessionToken() (string, error) {
	b := make([]byte, sessionTokenSize)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// GenerateRecoveryPhrase generates a random 6-word recovery phrase from the
// built-in wordlist. The phrase is shown to the user exactly once at setup.
func GenerateRecoveryPhrase() (string, error) {
	words := make([]string, recoveryWords)
	listLen := int64(len(wordlist))
	buf := make([]byte, 8)
	for i := range words {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("failed to generate recovery phrase: %w", err)
		}
		// Convert 8 random bytes to uint64, mod by wordlist length
		var n uint64
		for _, b := range buf {
			n = n<<8 | uint64(b)
		}
		words[i] = wordlist[n%uint64(listLen)]
	}
	return strings.Join(words, "-"), nil
}

// NormalizeRecoveryPhrase lowercases and trims a recovery phrase for comparison.
func NormalizeRecoveryPhrase(phrase string) string {
	return strings.ToLower(strings.TrimSpace(phrase))
}
