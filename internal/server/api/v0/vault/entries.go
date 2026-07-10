package v0_vault

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/autobutler-org/autobutler/internal/db"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/autobutler-org/autobutler/pkg/util/vaultcrypto"
	"github.com/gin-gonic/gin"
)

type entryListItem struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	URLHost   string `json:"urlHost"`
	FolderID  *int64 `json:"folderId"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type entryDetail struct {
	ID           int64         `json:"id"`
	Name         string        `json:"name"`
	URL          string        `json:"url"`
	URLHost      string        `json:"urlHost"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
	Notes        string        `json:"notes,omitempty"`
	TOTPSecret   string        `json:"totpSecret,omitempty"`
	CustomFields []customField `json:"customFields,omitempty"`
	FolderID     *int64        `json:"folderId"`
	CreatedAt    string        `json:"createdAt"`
	UpdatedAt    string        `json:"updatedAt"`
}

type createEntryRequest struct {
	Name         string        `json:"name" binding:"required"`
	URL          string        `json:"url"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
	Notes        string        `json:"notes"`
	TOTPSecret   string        `json:"totpSecret"`
	CustomFields []customField `json:"customFields"`
	FolderID     *int64        `json:"folderId"`
}

func nullableInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func fromNullInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

var listEntriesRoute = serverutil.ApiRoute(
	"GET", "/vault/entries", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		rows, err := deps.VaultDB().Queries.ListVaultEntries(c.Request.Context())
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("list entries: %w", err))
		}

		items := make([]entryListItem, 0, len(rows))
		for _, r := range rows {
			items = append(items, entryListItem{
				ID:        r.ID,
				Name:      r.Name,
				URLHost:   r.UrlHost,
				FolderID:  fromNullInt64(r.FolderID),
				CreatedAt: r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
				UpdatedAt: r.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			})
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"entries": items,
		})
	},
)

var getEntryRoute = serverutil.ApiRoute(
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

var createEntryRoute = serverutil.ApiRoute(
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

var updateEntryRoute = serverutil.ApiRoute(
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

var deleteEntryRoute = serverutil.ApiRoute(
	"DELETE", "/vault/entries/:id", func(c *gin.Context) *serverutil.Response {
		deps, key, errResp := requireUnlockedVault(c)
		if errResp != nil {
			return errResp
		}
		defer vaultcrypto.ZeroKey(key)

		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			return serverutil.BadRequest(fmt.Errorf("invalid id"))
		}

		if err := deps.VaultDB().Queries.DeleteVaultEntry(c.Request.Context(), id); err != nil {
			return serverutil.InternalServerError(fmt.Errorf("delete entry: %w", err))
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"deleted": true,
		})
	},
)
