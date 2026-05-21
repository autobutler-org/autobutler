package vaultcrypto

import (
	"bytes"
	"testing"
	"time"
)

func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt: %v", err)
	}
	if len(salt1) != SaltLen {
		t.Fatalf("salt length = %d, want %d", len(salt1), SaltLen)
	}

	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt: %v", err)
	}
	if bytes.Equal(salt1, salt2) {
		t.Fatal("two salts should not be equal")
	}
}

func TestDeriveKey_Deterministic(t *testing.T) {
	salt, _ := GenerateSalt()
	params := Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1}

	key1 := DeriveKey("password", salt, params)
	key2 := DeriveKey("password", salt, params)

	if !bytes.Equal(key1, key2) {
		t.Fatal("same inputs should produce the same key")
	}
	if len(key1) != KeyLen {
		t.Fatalf("key length = %d, want %d", len(key1), KeyLen)
	}
}

func TestDeriveKey_DifferentPasswords(t *testing.T) {
	salt, _ := GenerateSalt()
	params := Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1}

	key1 := DeriveKey("password1", salt, params)
	key2 := DeriveKey("password2", salt, params)

	if bytes.Equal(key1, key2) {
		t.Fatal("different passwords should produce different keys")
	}
}

func TestDeriveKey_DifferentSalts(t *testing.T) {
	salt1, _ := GenerateSalt()
	salt2, _ := GenerateSalt()
	params := Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1}

	key1 := DeriveKey("password", salt1, params)
	key2 := DeriveKey("password", salt2, params)

	if bytes.Equal(key1, key2) {
		t.Fatal("different salts should produce different keys")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := DeriveKey("test", []byte("saltsaltsaltsalt"), Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1})
	plaintext := []byte(`{"username":"alice","password":"s3cret"}`)

	ciphertext, nonce, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext should not equal plaintext")
	}

	decrypted, err := Decrypt(key, ciphertext, nonce)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestEncrypt_UniqueNonces(t *testing.T) {
	key := DeriveKey("test", []byte("saltsaltsaltsalt"), Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1})

	_, nonce1, _ := Encrypt(key, []byte("data"))
	_, nonce2, _ := Encrypt(key, []byte("data"))

	if bytes.Equal(nonce1, nonce2) {
		t.Fatal("nonces should be unique per encryption")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := DeriveKey("correct", []byte("saltsaltsaltsalt"), Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1})
	key2 := DeriveKey("wrong", []byte("saltsaltsaltsalt"), Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1})

	ciphertext, nonce, _ := Encrypt(key1, []byte("secret"))
	_, err := Decrypt(key2, ciphertext, nonce)

	if err == nil {
		t.Fatal("decryption with wrong key should fail")
	}
	if err != ErrDecryptionFailed {
		t.Fatalf("expected ErrDecryptionFailed, got: %v", err)
	}
}

func TestDecrypt_InvalidNonce(t *testing.T) {
	key := DeriveKey("test", []byte("saltsaltsaltsalt"), Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1})

	_, err := Decrypt(key, []byte("ciphertext"), []byte("short"))
	if err != ErrInvalidNonce {
		t.Fatalf("expected ErrInvalidNonce, got: %v", err)
	}
}

func TestVerificationBlob_CorrectPassword(t *testing.T) {
	key := DeriveKey("master", []byte("saltsaltsaltsalt"), Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1})

	blob, nonce, err := MakeVerificationBlob(key)
	if err != nil {
		t.Fatalf("MakeVerificationBlob: %v", err)
	}

	if !CheckVerificationBlob(key, blob, nonce) {
		t.Fatal("verification should pass with correct key")
	}
}

func TestVerificationBlob_WrongPassword(t *testing.T) {
	key1 := DeriveKey("correct", []byte("saltsaltsaltsalt"), Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1})
	key2 := DeriveKey("wrong", []byte("saltsaltsaltsalt"), Argon2Params{Memory: 1024, Iterations: 1, Parallelism: 1})

	blob, nonce, _ := MakeVerificationBlob(key1)

	if CheckVerificationBlob(key2, blob, nonce) {
		t.Fatal("verification should fail with wrong key")
	}
}

func TestZeroKey(t *testing.T) {
	key := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	ZeroKey(key)

	for i, b := range key {
		if b != 0 {
			t.Fatalf("key[%d] = %d, want 0", i, b)
		}
	}
}

func TestVaultSession_UnlockAndKey(t *testing.T) {
	s := NewVaultSession()
	key := []byte("12345678901234567890123456789012")

	if !s.IsLocked() {
		t.Fatal("new session should be locked")
	}

	s.Unlock(key, 5*time.Second)

	if s.IsLocked() {
		t.Fatal("session should be unlocked")
	}

	got, ok := s.Key()
	if !ok {
		t.Fatal("Key() should return true when unlocked")
	}
	if !bytes.Equal(got, key) {
		t.Fatal("Key() should return the stored key")
	}
}

func TestVaultSession_Lock(t *testing.T) {
	s := NewVaultSession()
	key := []byte("12345678901234567890123456789012")

	s.Unlock(key, 5*time.Second)
	s.Lock()

	if !s.IsLocked() {
		t.Fatal("session should be locked after Lock()")
	}

	_, ok := s.Key()
	if ok {
		t.Fatal("Key() should return false after Lock()")
	}
}

func TestVaultSession_Timeout(t *testing.T) {
	s := NewVaultSession()
	key := []byte("12345678901234567890123456789012")

	s.Unlock(key, 50*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	if !s.IsLocked() {
		t.Fatal("session should be locked after timeout")
	}
}

func TestVaultSession_Touch(t *testing.T) {
	s := NewVaultSession()
	key := []byte("12345678901234567890123456789012")

	s.Unlock(key, 100*time.Millisecond)
	time.Sleep(60 * time.Millisecond)
	s.Touch()
	time.Sleep(60 * time.Millisecond)

	if s.IsLocked() {
		t.Fatal("session should still be unlocked after Touch()")
	}
}

func TestVaultSession_KeyReturnsCopy(t *testing.T) {
	s := NewVaultSession()
	key := []byte("12345678901234567890123456789012")

	s.Unlock(key, 5*time.Second)

	got, _ := s.Key()
	got[0] = 0xFF

	got2, _ := s.Key()
	if got2[0] == 0xFF {
		t.Fatal("Key() should return a copy, not a reference to the internal key")
	}
}
