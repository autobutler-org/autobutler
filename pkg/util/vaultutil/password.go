package vaultutil

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const (
	lowerChars     = "abcdefghijklmnopqrstuvwxyz"
	upperChars     = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars     = "0123456789"
	symbolChars    = "!@#$%^&*()-_=+[]{}|;:,.<>?"
	ambiguousChars = "0O1lI"

	minGeneratedLength = 8
	maxGeneratedLength = 128
)

// GeneratePasswordParams describes the password to generate. Out-of-range
// lengths are clamped and a request with every character class switched off
// falls back to lowercase plus digits, so the generator always produces
// something usable rather than refusing.
type GeneratePasswordParams struct {
	Length         int
	Uppercase      bool
	Lowercase      bool
	Digits         bool
	Symbols        bool
	AvoidAmbiguous bool
}

// GeneratePasswordResult carries the generated password.
type GeneratePasswordResult struct {
	Password string
}

// GeneratePassword draws a password from the requested character classes using
// the cryptographic random source.
func GeneratePassword(params GeneratePasswordParams) (GeneratePasswordResult, error) {
	if params.Length < minGeneratedLength {
		params.Length = minGeneratedLength
	}
	if params.Length > maxGeneratedLength {
		params.Length = maxGeneratedLength
	}
	if !params.Uppercase && !params.Lowercase && !params.Digits && !params.Symbols {
		params.Lowercase = true
		params.Digits = true
	}

	var charset strings.Builder
	if params.Lowercase {
		charset.WriteString(lowerChars)
	}
	if params.Uppercase {
		charset.WriteString(upperChars)
	}
	if params.Digits {
		charset.WriteString(digitChars)
	}
	if params.Symbols {
		charset.WriteString(symbolChars)
	}

	pool := charset.String()
	if params.AvoidAmbiguous {
		var filtered strings.Builder
		for _, ch := range pool {
			if !strings.ContainsRune(ambiguousChars, ch) {
				filtered.WriteRune(ch)
			}
		}
		pool = filtered.String()
	}

	if len(pool) == 0 {
		return GeneratePasswordResult{}, ErrEmptyCharset
	}

	password, err := secureRandomString(pool, params.Length)
	if err != nil {
		return GeneratePasswordResult{}, fmt.Errorf("generate password: %w", err)
	}

	return GeneratePasswordResult{Password: password}, nil
}

// secureRandomString draws length characters uniformly from charset using
// crypto/rand — rejection-free because [rand.Int] already returns a uniform
// value below the limit.
func secureRandomString(charset string, length int) (string, error) {
	if length < 1 || length > maxGeneratedLength {
		return "", fmt.Errorf("length must be between 1 and %d", maxGeneratedLength)
	}
	limit := big.NewInt(int64(len(charset)))
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, limit)
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}
