package v0_vault

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"

	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

func getDeps(c *gin.Context) (deputil.Dependencies, *serverutil.Response) {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return nil, serverutil.InternalServerError(nil)
	}
	return deps, nil
}

func requireVaultAvailable(c *gin.Context) (deputil.Dependencies, *serverutil.Response) {
	deps, errResp := getDeps(c)
	if errResp != nil {
		return nil, errResp
	}
	if deps.VaultDB() == nil {
		return nil, &serverutil.Response{
			StatusCode:  503,
			ContentType: serverutil.ContentTypeJSON,
			Error:       fmt.Errorf("vault storage device is disconnected"),
		}
	}
	return deps, nil
}

func requireUnlockedVault(c *gin.Context) (deputil.Dependencies, []byte, *serverutil.Response) {
	deps, errResp := requireVaultAvailable(c)
	if errResp != nil {
		return nil, nil, errResp
	}

	key, ok := deps.VaultSession().Key()
	if !ok {
		reason := deps.VaultSession().LockReason()
		msg := "vault is locked"
		if reason != "" {
			msg = fmt.Sprintf("vault is locked: %s", reason)
		}
		return nil, nil, &serverutil.Response{
			StatusCode:  423,
			ContentType: serverutil.ContentTypeJSON,
			Error:       errors.New(msg),
		}
	}

	deps.VaultSession().Touch()
	return deps, key, nil
}

func extractURLHost(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// entryPayload is the decrypted JSON stored inside vault_entries.ciphertext.
type entryPayload struct {
	URL          string        `json:"url"`
	Username     string        `json:"username"`
	Password     string        `json:"password"`
	Notes        string        `json:"notes,omitempty"`
	TOTPSecret   string        `json:"totpSecret,omitempty"`
	CustomFields []customField `json:"customFields,omitempty"`
}

type customField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Hidden bool   `json:"hidden"`
}

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

type folderJSON struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ParentID  *int64 `json:"parentId"`
	SortOrder int64  `json:"sortOrder"`
	CreatedAt string `json:"createdAt"`
}
