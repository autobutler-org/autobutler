// Package shareutil manages public share links: tokenized, optionally
// password-protected, optionally expiring links to files or folders in the
// cirrus directory. Shares are persisted as a JSON file in the data directory
// (see AGENTS.md — the database is a last resort; a small rebuildable JSON
// file is sufficient here).
package shareutil

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/authutil"
	"github.com/autobutler-org/autobutler/pkg/util/storageutil"
)

const sharesFileName = "shares.json"

// Sentinel errors returned by Resolve. The HTTP layer maps these to status
// codes (404 / 410 / 401 / 403 respectively).
var (
	ErrNotFound         = errors.New("share not found")
	ErrExpired          = errors.New("share expired")
	ErrPasswordRequired = errors.New("share password required")
	ErrWrongPassword    = errors.New("wrong share password")
)

// Share is one public share link record.
type Share struct {
	ID           string     `json:"id"`
	Token        string     `json:"token"`
	FilePath     string     `json:"filePath"`
	DeviceSerial string     `json:"deviceSerial,omitempty"`
	PasswordHash string     `json:"passwordHash,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	AccessCount  int64      `json:"accessCount"`
	LastAccessAt *time.Time `json:"lastAccessAt,omitempty"`
}

// IsExpired reports whether the share's expiry (if any) has passed.
func (s *Share) IsExpired() bool {
	return s.ExpiresAt != nil && time.Now().After(*s.ExpiresAt)
}

// PasswordProtected reports whether the share requires a password.
func (s *Share) PasswordProtected() bool {
	return s.PasswordHash != ""
}

var (
	mu           sync.Mutex
	cached       []Share
	loaded       bool
	pathOverride string // set by ResetForTesting only
)

func sharesPath() string {
	if pathOverride != "" {
		return pathOverride
	}
	return filepath.Join(storageutil.GetDataDir(), sharesFileName)
}

// ResetForTesting points the store at a test-specific file and drops the
// in-process cache. Pass "" to restore the default path.
func ResetForTesting(path string) {
	mu.Lock()
	defer mu.Unlock()
	pathOverride = path
	cached = nil
	loaded = false
}

// load reads shares from disk into the cache. Caller must hold mu.
func load() error {
	if loaded {
		return nil
	}
	data, err := os.ReadFile(sharesPath())
	if os.IsNotExist(err) {
		cached = nil
		loaded = true
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read shares file: %w", err)
	}
	var shares []Share
	if err := json.Unmarshal(data, &shares); err != nil {
		return fmt.Errorf("failed to parse shares file: %w", err)
	}
	cached = shares
	loaded = true
	return nil
}

// save writes the cache to disk atomically (temp file + rename). The file is
// 0600 because it contains share tokens and password hashes. Caller must hold mu.
func save() error {
	path := sharesPath()
	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal shares: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("failed to write shares file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to replace shares file: %w", err)
	}
	return nil
}

// CreateShareParams contains parameters for creating a share link.
type CreateShareParams struct {
	// FilePath is the cirrus-relative path of the file or folder to share.
	FilePath     string
	DeviceSerial string
	// Password, when non-empty, protects the share (stored as a bcrypt hash).
	Password string
	// ExpiresAt, when non-nil, is the moment the share stops working.
	ExpiresAt *time.Time
}

// CreateShareResult contains the newly created share.
type CreateShareResult struct {
	Share Share
}

// Create generates a new share link. The caller is responsible for having
// validated that FilePath exists (see StorageService.StatFile).
func Create(params CreateShareParams) (*CreateShareResult, error) {
	if params.FilePath == "" {
		return nil, fmt.Errorf("filePath is required")
	}

	token, err := authutil.GenerateSessionToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate share token: %w", err)
	}
	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, fmt.Errorf("failed to generate share id: %w", err)
	}

	share := Share{
		ID:           hex.EncodeToString(idBytes),
		Token:        token,
		FilePath:     params.FilePath,
		DeviceSerial: params.DeviceSerial,
		CreatedAt:    time.Now().UTC(),
		ExpiresAt:    params.ExpiresAt,
	}
	if params.Password != "" {
		hash, err := authutil.HashPassword(params.Password)
		if err != nil {
			return nil, err
		}
		share.PasswordHash = hash
	}

	mu.Lock()
	defer mu.Unlock()
	if err := load(); err != nil {
		return nil, err
	}
	cached = append(cached, share)
	if err := save(); err != nil {
		return nil, err
	}
	return &CreateShareResult{Share: share}, nil
}

// List returns all shares, newest first.
func List() ([]Share, error) {
	mu.Lock()
	defer mu.Unlock()
	if err := load(); err != nil {
		return nil, err
	}
	out := make([]Share, len(cached))
	copy(out, cached)
	// Newest first — cached is append-ordered.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// Delete removes a share by ID. Returns ErrNotFound if no share has that ID.
func Delete(id string) error {
	mu.Lock()
	defer mu.Unlock()
	if err := load(); err != nil {
		return err
	}
	for i, s := range cached {
		if s.ID == id {
			cached = append(cached[:i], cached[i+1:]...)
			return save()
		}
	}
	return ErrNotFound
}

// Resolve looks up a share by token and validates expiry and password.
// On success it records the access (count + timestamp) and returns a copy of
// the share. Password checking uses bcrypt via authutil.
func Resolve(token, password string) (*Share, error) {
	if token == "" {
		return nil, ErrNotFound
	}
	mu.Lock()
	defer mu.Unlock()
	if err := load(); err != nil {
		return nil, err
	}
	for i := range cached {
		s := &cached[i]
		if s.Token != token {
			continue
		}
		if s.IsExpired() {
			return nil, ErrExpired
		}
		if s.PasswordProtected() {
			if password == "" {
				return nil, ErrPasswordRequired
			}
			if !authutil.CheckPassword(password, s.PasswordHash) {
				return nil, ErrWrongPassword
			}
		}
		now := time.Now().UTC()
		s.AccessCount++
		s.LastAccessAt = &now
		// Best-effort: access bookkeeping should never block a download.
		if err := save(); err != nil {
			slog.Warn("shareutil: failed to record share access", "err", err)
		}
		out := *s
		return &out, nil
	}
	return nil, ErrNotFound
}

// Peek looks up a share by token without password validation or access
// bookkeeping. Used by the public info endpoint, which must be able to tell
// a client that a password is required before it has one. Expired shares
// still return ErrExpired so links die consistently across endpoints.
func Peek(token string) (*Share, error) {
	if token == "" {
		return nil, ErrNotFound
	}
	mu.Lock()
	defer mu.Unlock()
	if err := load(); err != nil {
		return nil, err
	}
	for i := range cached {
		if cached[i].Token == token {
			if cached[i].IsExpired() {
				return nil, ErrExpired
			}
			out := cached[i]
			return &out, nil
		}
	}
	return nil, ErrNotFound
}
