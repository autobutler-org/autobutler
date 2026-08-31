package vaultutil

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
)

// Import formats the parser understands. FormatAuto asks [DetectFormat] to
// pick one from the bytes.
const (
	FormatAuto      = "auto"
	FormatJSON      = "json"
	FormatBitwarden = "bitwarden"
	FormatCSV       = "csv"
)

// ImportEntry is one login recovered from an export file, before it is
// encrypted into the vault.
type ImportEntry struct {
	Name       string
	URL        string
	Username   string
	Password   string
	Notes      string
	TOTPSecret string
	Folder     string
}

// ImportParams imports an export file from Quark or another password manager.
type ImportParams struct {
	// VaultDB is the vault database. The whole import runs in one
	// transaction on it, so a failure part way through imports nothing.
	VaultDB *db.DatabaseSqlc
	// Key is the unlocked vault key each imported entry is sealed under.
	Key []byte
	// Data is the raw export file.
	Data []byte
	// Format is one of the Format* constants. FormatAuto sniffs the bytes.
	Format string
}

// ImportResult counts what the import did. Errors holds one line per row the
// import could not take, so a mostly-good file still lands its good rows.
type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors,omitempty"`
}

// Import parses an export file and writes its entries into the vault inside a
// single transaction. Entries matching an existing name+host are skipped
// rather than duplicated, and folders named in the file are created on demand.
//
// It returns [ErrUnsupportedImportFormat] for a format it has no reader for;
// the caller names the format in the copy it renders.
// MaxImportBytes caps an import file. Detection and every parser here work on
// the whole document, so this is content that genuinely cannot be streamed —
// which makes the bound the caller's job rather than an optional extra. A real
// vault export of tens of thousands of entries is a few megabytes; this leaves
// an order of magnitude of headroom and still refuses a body that would
// otherwise be allocated in full (#1723).
const MaxImportBytes int64 = 32 * 1024 * 1024

// ErrImportTooLarge reports an import file over [MaxImportBytes].
var ErrImportTooLarge = fmt.Errorf("import file exceeds the maximum of %d bytes", MaxImportBytes)

// ReadImport reads an import file from r, refusing anything over
// [MaxImportBytes]. It reads one byte past the cap so an oversized file is
// rejected rather than silently truncated into a partial import.
func ReadImport(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxImportBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxImportBytes {
		return nil, ErrImportTooLarge
	}
	return data, nil
}

func Import(ctx context.Context, params ImportParams) (ImportResult, error) {
	format := params.Format
	if format == FormatAuto {
		format = DetectFormat(params.Data)
	}

	var entries []ImportEntry
	var parseErrors []string

	switch format {
	case FormatJSON:
		entries, parseErrors = ParseQuarkJSON(params.Data)
	case FormatBitwarden:
		entries, parseErrors = ParseBitwardenCSV(params.Data)
	case FormatCSV:
		entries, parseErrors = ParseGenericCSV(params.Data)
	default:
		return ImportResult{}, ErrUnsupportedImportFormat
	}

	tx, err := params.VaultDB.Db.BeginTx(ctx, nil)
	if err != nil {
		return ImportResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := params.VaultDB.Queries.WithTx(tx)

	existingEntries, err := qtx.ListVaultEntries(ctx)
	if err != nil {
		return ImportResult{}, fmt.Errorf("list entries: %w", err)
	}
	dedupSet := make(map[string]bool)
	for _, e := range existingEntries {
		dedupSet[dedupKey(e.Name, e.UrlHost)] = true
	}

	existingFolders, err := qtx.ListVaultFolders(ctx)
	if err != nil {
		return ImportResult{}, fmt.Errorf("list folders: %w", err)
	}
	folderMap := make(map[string]int64)
	for _, f := range existingFolders {
		folderMap[f.Name] = f.ID
	}

	result := ImportResult{Errors: parseErrors}

	for _, entry := range entries {
		host := HostFromURL(entry.URL)
		if dedupSet[dedupKey(entry.Name, host)] {
			result.Skipped++
			continue
		}

		var folderID sql.NullInt64
		if entry.Folder != "" {
			if id, ok := folderMap[entry.Folder]; ok {
				folderID = sql.NullInt64{Int64: id, Valid: true}
			} else {
				f, err := qtx.CreateVaultFolder(ctx, db.CreateVaultFolderParams{
					Name: entry.Folder,
				})
				if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("create folder %q: %v", entry.Folder, err))
					continue
				}
				folderMap[entry.Folder] = f.ID
				folderID = sql.NullInt64{Int64: f.ID, Valid: true}
			}
		}

		payload := EntryPayload{
			URL:        entry.URL,
			Username:   entry.Username,
			Password:   entry.Password,
			Notes:      entry.Notes,
			TOTPSecret: entry.TOTPSecret,
		}
		plaintext, _ := json.Marshal(payload)
		ct, nonce, err := vaultcrypto.Encrypt(params.Key, plaintext)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("encrypt %q: %v", entry.Name, err))
			continue
		}

		_, err = qtx.CreateVaultEntry(ctx, db.CreateVaultEntryParams{
			Name:       entry.Name,
			UrlHost:    host,
			FolderID:   folderID,
			Ciphertext: ct,
			Nonce:      nonce,
		})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("create %q: %v", entry.Name, err))
			continue
		}

		dedupSet[dedupKey(entry.Name, host)] = true
		result.Imported++
	}

	if err := tx.Commit(); err != nil {
		return ImportResult{}, fmt.Errorf("commit: %w", err)
	}

	return result, nil
}

// dedupKey identifies an entry by name and host, case-insensitively, so a
// re-import of the same export does not double every login.
func dedupKey(name, urlHost string) string {
	return strings.ToLower(name) + "\x00" + strings.ToLower(urlHost)
}

// DetectFormat guesses an export file's format from its first bytes: JSON by
// its opening brace or bracket, Bitwarden by its distinctive CSV header, and
// generic CSV for everything else.
func DetectFormat(data []byte) string {
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		return FormatJSON
	}
	firstLine := trimmed
	if idx := strings.Index(trimmed, "\n"); idx > 0 {
		firstLine = trimmed[:idx]
	}
	lower := strings.ToLower(firstLine)
	if strings.Contains(lower, "login_uri") || strings.Contains(lower, "login_username") {
		return FormatBitwarden
	}
	return FormatCSV
}

// ParseQuarkJSON reads Quark's own export, in either its object form or a bare
// array of entries.
func ParseQuarkJSON(data []byte) ([]ImportEntry, []string) {
	var export ExportJSON
	if err := json.Unmarshal(data, &export); err != nil {
		var arr []ExportEntry
		if err2 := json.Unmarshal(data, &arr); err2 != nil {
			return nil, []string{fmt.Sprintf("invalid JSON: %v", err)}
		}
		export.Entries = arr
	}

	entries := make([]ImportEntry, 0, len(export.Entries))
	for _, e := range export.Entries {
		entries = append(entries, ImportEntry{
			Name:       e.Name,
			URL:        e.URL,
			Username:   e.Username,
			Password:   e.Password,
			Notes:      e.Notes,
			TOTPSecret: e.TOTPSecret,
			Folder:     e.FolderName,
		})
	}
	return entries, nil
}

// ParseBitwardenCSV reads a Bitwarden CSV export, whose column names are
// specific enough to map directly.
func ParseBitwardenCSV(data []byte) ([]ImportEntry, []string) {
	records, errs := readCSV(data)
	if errs != nil {
		return nil, errs
	}

	idx := csvHeaderIndex(records[0])

	var entries []ImportEntry
	var errors []string

	for i, row := range records[1:] {
		e := ImportEntry{
			Name:       csvCol(row, idx, "name"),
			URL:        csvCol(row, idx, "login_uri"),
			Username:   csvCol(row, idx, "login_username"),
			Password:   csvCol(row, idx, "login_password"),
			Notes:      csvCol(row, idx, "notes"),
			TOTPSecret: csvCol(row, idx, "login_totp"),
			Folder:     csvCol(row, idx, "folder"),
		}
		if e.Name == "" && e.URL == "" {
			errors = append(errors, fmt.Sprintf("Row %d: missing name and URL", i+2))
			continue
		}
		if e.Name == "" {
			e.Name = HostFromURL(e.URL)
			if e.Name == "" {
				e.Name = e.URL
			}
		}
		entries = append(entries, e)
	}
	return entries, errors
}

// ParseGenericCSV reads a CSV from any of the browsers and managers that
// export one, accepting the several spellings each column goes by.
func ParseGenericCSV(data []byte) ([]ImportEntry, []string) {
	records, errs := readCSV(data)
	if errs != nil {
		return nil, errs
	}

	idx := csvHeaderIndex(records[0])

	var entries []ImportEntry
	var errors []string

	for i, row := range records[1:] {
		e := ImportEntry{
			Name:     csvColAny(row, idx, "name", "title"),
			URL:      csvColAny(row, idx, "url", "website", "uri", "login_uri"),
			Username: csvColAny(row, idx, "username", "user", "login", "login_username"),
			Password: csvColAny(row, idx, "password", "pass", "login_password"),
			Notes:    csvColAny(row, idx, "notes", "note", "comments"),
		}
		if e.Password == "" {
			errors = append(errors, fmt.Sprintf("Row %d: missing password", i+2))
			continue
		}
		if e.Name == "" {
			e.Name = HostFromURL(e.URL)
			if e.Name == "" {
				e.Name = fmt.Sprintf("Entry %d", i+2)
			}
		}
		entries = append(entries, e)
	}
	return entries, errors
}

// readCSV parses the whole file and rejects one with no data rows. A non-nil
// second return means the file is unusable and the first is empty.
func readCSV(data []byte) ([][]string, []string) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, []string{fmt.Sprintf("CSV parse error: %v", err)}
	}
	if len(records) < 2 {
		return nil, []string{"CSV has no data rows"}
	}
	return records, nil
}

// csvHeaderIndex maps lower-cased column names to their position.
func csvHeaderIndex(header []string) map[string]int {
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return idx
}

func csvCol(row []string, idx map[string]int, key string) string {
	if i, ok := idx[key]; ok && i < len(row) {
		return strings.TrimSpace(row[i])
	}
	return ""
}

func csvColAny(row []string, idx map[string]int, keys ...string) string {
	for _, k := range keys {
		if v := csvCol(row, idx, k); v != "" {
			return v
		}
	}
	return ""
}
