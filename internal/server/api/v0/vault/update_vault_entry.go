package v0_vault

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/autobutler-org/quark/pkg/util/vaultcrypto"
	"github.com/gin-gonic/gin"
)

var updateVaultEntryRoute = serverutil.ApiRoute(
	"PUT", "/vault/entries/:id", func(c *gin.Context) *serverutil.Response {
		deps, key, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}
		defer vaultcrypto.ZeroKey(key)

		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid id"))
		}

		ctx := c.Request.Context()

		if _, err := deps.VaultDB().Queries.GetVaultEntry(ctx, id); errors.Is(err, sql.ErrNoRows) {
			return serverutil.NotFound(fmt.Errorf("entry not found"))
		} else if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("get entry: %w", err))
		}

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

		if err := deps.VaultDB().Queries.UpdateVaultEntry(ctx, db.UpdateVaultEntryParams{
			Name:       req.Name,
			UrlHost:    extractURLHost(req.URL),
			FolderID:   nullableInt64(req.FolderID),
			Ciphertext: ciphertext,
			Nonce:      nonce,
			ID:         id,
		}); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("update entry: %w", err))
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"id": id,
		})
	},
)
