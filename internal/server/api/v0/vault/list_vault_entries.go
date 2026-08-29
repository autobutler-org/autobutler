package v0_vault

import (
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

var listVaultEntriesRoute = serverutil.ApiRoute(
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
