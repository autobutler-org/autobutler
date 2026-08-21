package vaultcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	KeyLen   = 32 // AES-256
	SaltLen  = 16
	NonceLen = 12 // GCM standard nonce size
)

var (
	ErrDecryptionFailed = errors.New("decryption failed: ciphertext is invalid or key is wrong")
	ErrInvalidNonce     = errors.New("nonce must be 12 bytes")
)

// verificationPlaintext is the known string we encrypt during vault setup.
// On unlock we decrypt the stored blob and compare to this value.
// This is intentionally not secret — security comes from the AES-256-GCM
// encryption, not from the plaintext being unknown.
var verificationPlaintext = []byte("quark-vault-ok")

// Argon2Params holds tunable parameters for Argon2id key derivation.
type Argon2Params struct {
	Memory      uint32 // KiB (default 65536 = 64MB)
	Iterations  uint32 // time cost (default 3)
	Parallelism uint8  // threads (default 4)
}

// DefaultParams returns recommended Argon2id parameters for vault key derivation.
func DefaultParams() Argon2Params {
	return Argon2Params{
		Memory:      65536,
		Iterations:  3,
		Parallelism: 4,
	}
}

// DeriveKey derives a 256-bit key from a master password and salt using Argon2id.
func DeriveKey(password string, salt []byte, params Argon2Params) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		KeyLen,
	)
}

// GenerateSalt returns 16 cryptographically random bytes suitable as an Argon2 salt.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}
	return salt, nil
}

// Encrypt encrypts plaintext with AES-256-GCM using the given key.
// Returns ciphertext and a randomly generated nonce.
func Encrypt(key, plaintext []byte) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce = make([]byte, NonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// Decrypt decrypts ciphertext with AES-256-GCM using the given key and nonce.
func Decrypt(key, ciphertext, nonce []byte) ([]byte, error) {
	if len(nonce) != NonceLen {
		return nil, ErrInvalidNonce
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}
	return plaintext, nil
}

// MakeVerificationBlob encrypts the known verification plaintext with the given key.
// Store the returned ciphertext and nonce in vault_config to verify the master password on unlock.
func MakeVerificationBlob(key []byte) (ciphertext, nonce []byte, err error) {
	return Encrypt(key, verificationPlaintext)
}

// CheckVerificationBlob decrypts the stored verification blob and checks it matches
// the expected plaintext. Returns true if the key (and thus the master password) is correct.
func CheckVerificationBlob(key, ciphertext, nonce []byte) bool {
	plaintext, err := Decrypt(key, ciphertext, nonce)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(plaintext, verificationPlaintext) == 1
}

// ZeroKey securely zeroes a key slice in memory.
func ZeroKey(key []byte) {
	for i := range key {
		key[i] = 0
	}
}
