package v0_vault

import (
	"encoding/csv"
	"log/slog"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/autobutler-org/quark/pkg/util/vaultutil"
	"github.com/gin-gonic/gin"
)

var exportVaultRoute = serverutil.ApiRoute(
	"GET", "/vault/export", func(c *gin.Context) *serverutil.Response {
		deps, key, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}
		defer vaultcrypto.ZeroKey(key)

		format := c.DefaultQuery("format", "json")

		result, err := vaultutil.Export(c.Request.Context(), vaultutil.ExportParams{
			Queries: deps.VaultDB().Queries,
			Key:     key,
		})
		if err != nil {
			return serverutil.InternalServerError(err)
		}

		c.Header("Cache-Control", "no-store")
		c.Header("X-Content-Type-Options", "nosniff")

		switch format {
		case "csv":
			records := make([][]string, 0, len(result.Entries)+1)
			records = append(records, []string{"name", "url", "username", "password", "notes", "totp_secret", "folder"})
			for _, e := range result.Entries {
				records = append(records, []string{e.Name, e.URL, e.Username, e.Password, e.Notes, e.TOTPSecret, e.FolderName})
			}

			c.Header("Content-Disposition", "attachment; filename=quark_vault.csv")
			c.Header("Content-Type", "text/csv; charset=utf-8")
			c.Status(200)

			// Written straight onto the response. It used to go into a
			// strings.Builder and then through []byte(buf.String()), which made
			// two full copies of the export before anything was sent (#1723).
			// The cost is that a mid-write failure lands after the headers, so
			// it can only be logged.
			if err := csv.NewWriter(c.Writer).WriteAll(records); err != nil {
				slog.Error("vault export: csv stream write failed", "err", err)
			}
			return nil

		default:
			c.Header("Content-Disposition", "attachment; filename=quark_vault.json")
			return serverutil.Ok().WithData(vaultutil.ExportJSON{
				Entries: result.Entries,
				Folders: result.FolderNames,
			})
		}
	},
)
