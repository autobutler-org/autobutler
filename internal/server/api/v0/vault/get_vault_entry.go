package v0_vault

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/gin-gonic/gin"
)

var getVaultEntryRoute = serverutil.ApiRoute(
	"GET", "/vault/entries/:id", func(c *gin.Context) *serverutil.Response {
		deps, key, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}
		defer vaultcrypto.ZeroKey(key)

		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid id"))
		}

		entry, err := deps.VaultDB().Queries.GetVaultEntry(c.Request.Context(), id)
		if errors.Is(err, sql.ErrNoRows) {
			return serverutil.NotFound(fmt.Errorf("entry not found"))
		}
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("get entry: %w", err))
		}

		plaintext, err := vaultcrypto.Decrypt(key, entry.Ciphertext, entry.Nonce)
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("decrypt entry: %w", err))
		}

		var payload entryPayload
		if err := json.Unmarshal(plaintext, &payload); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("unmarshal entry: %w", err))
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(entryDetail{
			ID:           entry.ID,
			Name:         entry.Name,
			URL:          payload.URL,
			URLHost:      entry.UrlHost,
			Username:     payload.Username,
			Password:     payload.Password,
			Notes:        payload.Notes,
			TOTPSecret:   payload.TOTPSecret,
			CustomFields: payload.CustomFields,
			FolderID:     fromNullInt64(entry.FolderID),
			CreatedAt:    entry.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			UpdatedAt:    entry.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	},
)
