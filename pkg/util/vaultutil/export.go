package vaultutil

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
)

// ExportEntry is one fully decrypted entry in a plaintext export. This is the
// shape Quark writes and the shape [ParseQuarkJSON] reads back, so an export
// round-trips through an import.
type ExportEntry struct {
	Name         string        `json:"name"`
	URL          string        `json:"url"`
	URLHost      string        `json:"urlHost"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
	Notes        string        `json:"notes,omitempty"`
	TOTPSecret   string        `json:"totpSecret,omitempty"`
	CustomFields []CustomField `json:"customFields,omitempty"`
	FolderName   string        `json:"folderName,omitempty"`
}

// ExportJSON is the document a JSON export serializes to.
type ExportJSON struct {
	Entries []ExportEntry `json:"entries"`
	Folders []string      `json:"folders"`
}

// ExportParams decrypts the whole vault for a plaintext export.
type ExportParams struct {
	Queries *db.Queries
	// Key is the unlocked vault key.
	Key []byte
}

// ExportResult carries every entry in the clear plus the folder names, ready
// to be serialized as JSON or CSV.
type ExportResult struct {
	Entries     []ExportEntry
	FolderNames []string
}

// Export reads and decrypts every entry in the vault. It is the one operation
// that deliberately produces plaintext secrets, so its caller is responsible
// for the no-store headers that keep the response out of caches.
func Export(ctx context.Context, params ExportParams) (ExportResult, error) {
	folders, err := params.Queries.ListVaultFolders(ctx)
	if err != nil {
		return ExportResult{}, fmt.Errorf("list folders: %w", err)
	}
	folderMap := make(map[int64]string)
	folderNames := make([]string, 0, len(folders))
	for _, f := range folders {
		folderMap[f.ID] = f.Name
		folderNames = append(folderNames, f.Name)
	}

	entries, err := params.Queries.ListAllVaultEntriesForReEncrypt(ctx)
	if err != nil {
		return ExportResult{}, fmt.Errorf("list entries: %w", err)
	}

	var exported []ExportEntry
	for _, e := range entries {
		full, err := params.Queries.GetVaultEntry(ctx, e.ID)
		if err != nil {
			return ExportResult{}, fmt.Errorf("get entry %d: %w", e.ID, err)
		}

		plaintext, err := vaultcrypto.Decrypt(params.Key, full.Ciphertext, full.Nonce)
		if err != nil {
			return ExportResult{}, fmt.Errorf("decrypt entry %d: %w", e.ID, err)
		}

		var payload EntryPayload
		if err := json.Unmarshal(plaintext, &payload); err != nil {
			return ExportResult{}, fmt.Errorf("unmarshal entry %d: %w", e.ID, err)
		}

		folderName := ""
		if full.FolderID.Valid {
			folderName = folderMap[full.FolderID.Int64]
		}

		exported = append(exported, ExportEntry{
			Name:         full.Name,
			URL:          payload.URL,
			URLHost:      full.UrlHost,
			Username:     payload.Username,
			Password:     payload.Password,
			Notes:        payload.Notes,
			TOTPSecret:   payload.TOTPSecret,
			CustomFields: payload.CustomFields,
			FolderName:   folderName,
		})
	}

	return ExportResult{Entries: exported, FolderNames: folderNames}, nil
}
