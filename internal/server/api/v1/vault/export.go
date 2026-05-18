package v1_vault

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/vaultcrypto"
	"github.com/gin-gonic/gin"
)

type exportEntry struct {
	Name         string        `json:"name"`
	URL          string        `json:"url"`
	URLHost      string        `json:"urlHost"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
	Notes        string        `json:"notes,omitempty"`
	TOTPSecret   string        `json:"totpSecret,omitempty"`
	CustomFields []customField `json:"customFields,omitempty"`
	FolderName   string        `json:"folderName,omitempty"`
}

type exportJSON struct {
	Entries []exportEntry `json:"entries"`
	Folders []string      `json:"folders"`
}

var exportVaultRoute = serverutil.ApiRoute(
	"GET", "/vault/export", func(c *gin.Context) *serverutil.Response {
		deps, key, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}
		defer vaultcrypto.ZeroKey(key)

		ctx := c.Request.Context()
		format := c.DefaultQuery("format", "json")

		folders, err := deps.Database().Queries.ListVaultFolders(ctx)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("list folders: %w", err))
		}
		folderMap := make(map[int64]string)
		folderNames := make([]string, 0, len(folders))
		for _, f := range folders {
			folderMap[f.ID] = f.Name
			folderNames = append(folderNames, f.Name)
		}

		entries, err := deps.Database().Queries.ListAllVaultEntriesForReEncrypt(ctx)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("list entries: %w", err))
		}

		var exported []exportEntry
		for _, e := range entries {
			full, err := deps.Database().Queries.GetVaultEntry(ctx, e.ID)
			if err != nil {
				return serverutil.InternalServerError(fmt.Errorf("get entry %d: %w", e.ID, err))
			}

			plaintext, err := vaultcrypto.Decrypt(key, full.Ciphertext, full.Nonce)
			if err != nil {
				return serverutil.InternalServerError(fmt.Errorf("decrypt entry %d: %w", e.ID, err))
			}

			var payload entryPayload
			if err := json.Unmarshal(plaintext, &payload); err != nil {
				return serverutil.InternalServerError(fmt.Errorf("unmarshal entry %d: %w", e.ID, err))
			}

			folderName := ""
			if full.FolderID.Valid {
				folderName = folderMap[full.FolderID.Int64]
			}

			exported = append(exported, exportEntry{
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

		c.Header("Cache-Control", "no-store")
		c.Header("X-Content-Type-Options", "nosniff")

		switch format {
		case "csv":
			var buf strings.Builder
			w := csv.NewWriter(&buf)
			w.Write([]string{"name", "url", "username", "password", "notes", "totp_secret", "folder"})
			for _, e := range exported {
				w.Write([]string{e.Name, e.URL, e.Username, e.Password, e.Notes, e.TOTPSecret, e.FolderName})
			}
			w.Flush()

			c.Header("Content-Disposition", "attachment; filename=autobutler_vault.csv")
			c.Data(200, "text/csv; charset=utf-8", []byte(buf.String()))
			return nil

		default:
			c.Header("Content-Disposition", "attachment; filename=autobutler_vault.json")
			return serverutil.Ok().WithData(exportJSON{
				Entries: exported,
				Folders: folderNames,
			})
		}
	},
)
