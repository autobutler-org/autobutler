package v0_vault

import (
	"encoding/json"
	"fmt"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/gin-gonic/gin"
)

var createVaultEntryRoute = serverutil.ApiRoute(
	"POST", "/vault/entries", func(c *gin.Context) *serverutil.Response {
		deps, key, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}
		defer vaultcrypto.ZeroKey(key)

		var req createEntryRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid request: %w", err))
		}

		payload := entryPayload{
			URL:          req.URL,
			Username:     req.Username,
			Password:     req.Password,
			Notes:        req.Notes,
			TOTPSecret:   req.TOTPSecret,
			CustomFields: req.CustomFields,
		}

		plaintext, err := json.Marshal(payload)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("marshal entry: %w", err))
		}

		ciphertext, nonce, err := vaultcrypto.Encrypt(key, plaintext)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("encrypt entry: %w", err))
		}

		entry, err := deps.VaultDB().Queries.CreateVaultEntry(c.Request.Context(), db.CreateVaultEntryParams{
			Name:       req.Name,
			UrlHost:    extractURLHost(req.URL),
			FolderID:   nullableInt64(req.FolderID),
			Ciphertext: ciphertext,
			Nonce:      nonce,
		})
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("create entry: %w", err))
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(entryDetail{
			ID:           entry.ID,
			Name:         entry.Name,
			URL:          req.URL,
			URLHost:      entry.UrlHost,
			Username:     req.Username,
			Password:     req.Password,
			Notes:        req.Notes,
			TOTPSecret:   req.TOTPSecret,
			CustomFields: req.CustomFields,
			FolderID:     fromNullInt64(entry.FolderID),
			CreatedAt:    entry.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			UpdatedAt:    entry.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	},
)
