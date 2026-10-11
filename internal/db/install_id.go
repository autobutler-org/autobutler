package db

import (
	"crypto/rand"
	"encoding/hex"
)

// NewInstallID returns a fresh install id for RotateInstallID: 16 random bytes
// as 32 lowercase hex digits, the shape migration 029 seeds. It is made here
// rather than in the query so that no engine needs a random-bytes function.
func NewInstallID() string {
	var raw [16]byte
	// crypto/rand.Read never returns an error: it aborts the program instead.
	_, _ = rand.Read(raw[:])
	return hex.EncodeToString(raw[:])
}
