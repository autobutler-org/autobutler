package v0_vault

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/vaultcrypto"
	"github.com/gin-gonic/gin"
)

type importResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors,omitempty"`
}

var importVaultRoute = serverutil.ApiRoute(
	"POST", "/vault/import", func(c *gin.Context) *serverutil.Response {
		deps, key, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}
		defer vaultcrypto.ZeroKey(key)

		file, _, err := c.Request.FormFile("file")
		if err != nil {
			return serverutil.BadRequest(fmt.Errorf("file required: %w", err))
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("read file: %w", err))
		}

		format := c.DefaultPostForm("format", "auto")
		if format == "auto" {
			format = detectFormat(data)
		}

		var entries []importEntry
		var parseErrors []string

		switch format {
		case "json":
			entries, parseErrors = parseAutoButlerJSON(data)
		case "bitwarden":
			entries, parseErrors = parseBitwardenCSV(data)
		case "csv":
			entries, parseErrors = parseGenericCSV(data)
		default:
			return serverutil.BadRequest(fmt.Errorf("unsupported format: %s", format))
		}

		ctx := c.Request.Context()

		tx, err := deps.VaultDB().Db.BeginTx(ctx, nil)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("begin tx: %w", err))
		}
		defer tx.Rollback()

		qtx := deps.VaultDB().Queries.WithTx(tx)

		existingEntries, err := qtx.ListVaultEntries(ctx)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("list entries: %w", err))
		}
		dedupSet := make(map[string]bool)
		for _, e := range existingEntries {
			dedupSet[dedupKey(e.Name, e.UrlHost)] = true
		}

		existingFolders, err := qtx.ListVaultFolders(ctx)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("list folders: %w", err))
		}
		folderMap := make(map[string]int64)
		for _, f := range existingFolders {
			folderMap[f.Name] = f.ID
		}

		result := importResult{Errors: parseErrors}

		for _, entry := range entries {
			host := hostFromURL(entry.URL)
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

			payload := entryPayload{
				URL:        entry.URL,
				Username:   entry.Username,
				Password:   entry.Password,
				Notes:      entry.Notes,
				TOTPSecret: entry.TOTPSecret,
			}
			plaintext, _ := json.Marshal(payload)
			ct, nonce, err := vaultcrypto.Encrypt(key, plaintext)
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
			return serverutil.InternalServerError(fmt.Errorf("commit: %w", err))
		}

		return serverutil.Ok().WithData(result)
	},
)

type importEntry struct {
	Name       string
	URL        string
	Username   string
	Password   string
	Notes      string
	TOTPSecret string
	Folder     string
}

func dedupKey(name, urlHost string) string {
	return strings.ToLower(name) + "\x00" + strings.ToLower(urlHost)
}

func hostFromURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func detectFormat(data []byte) string {
	trimmed := strings.TrimSpace(string(data))
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') {
		return "json"
	}
	firstLine := trimmed
	if idx := strings.Index(trimmed, "\n"); idx > 0 {
		firstLine = trimmed[:idx]
	}
	lower := strings.ToLower(firstLine)
	if strings.Contains(lower, "login_uri") || strings.Contains(lower, "login_username") {
		return "bitwarden"
	}
	return "csv"
}

func parseAutoButlerJSON(data []byte) ([]importEntry, []string) {
	var export exportJSON
	if err := json.Unmarshal(data, &export); err != nil {
		var arr []exportEntry
		if err2 := json.Unmarshal(data, &arr); err2 != nil {
			return nil, []string{fmt.Sprintf("invalid JSON: %v", err)}
		}
		export.Entries = arr
	}

	entries := make([]importEntry, 0, len(export.Entries))
	for _, e := range export.Entries {
		entries = append(entries, importEntry{
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

func parseBitwardenCSV(data []byte) ([]importEntry, []string) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, []string{fmt.Sprintf("CSV parse error: %v", err)}
	}
	if len(records) < 2 {
		return nil, []string{"CSV has no data rows"}
	}

	header := records[0]
	idx := make(map[string]int)
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var entries []importEntry
	var errors []string

	for i, row := range records[1:] {
		e := importEntry{
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
			e.Name = hostFromURL(e.URL)
			if e.Name == "" {
				e.Name = e.URL
			}
		}
		entries = append(entries, e)
	}
	return entries, errors
}

func parseGenericCSV(data []byte) ([]importEntry, []string) {
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, []string{fmt.Sprintf("CSV parse error: %v", err)}
	}
	if len(records) < 2 {
		return nil, []string{"CSV has no data rows"}
	}

	header := records[0]
	idx := make(map[string]int)
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var entries []importEntry
	var errors []string

	for i, row := range records[1:] {
		e := importEntry{
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
			e.Name = hostFromURL(e.URL)
			if e.Name == "" {
				e.Name = fmt.Sprintf("Entry %d", i+2)
			}
		}
		entries = append(entries, e)
	}
	return entries, errors
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
