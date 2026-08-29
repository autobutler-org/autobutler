package v0_vault

import (
	"fmt"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

var listVaultFoldersRoute = serverutil.ApiRoute(
	"GET", "/vault/folders", func(c *gin.Context) *serverutil.Response {
		deps, errResp := getDeps(c)
		if errResp != nil {
			return errResp
		}

		rows, err := deps.VaultDB().Queries.ListVaultFolders(c.Request.Context())
		if err != nil {
			return serverutil.InternalServerError(fmt.Errorf("list folders: %w", err))
		}

		folders := make([]folderJSON, 0, len(rows))
		for _, r := range rows {
			folders = append(folders, folderJSON{
				ID:        r.ID,
				Name:      r.Name,
				ParentID:  fromNullInt64(r.ParentID),
				SortOrder: r.SortOrder,
				CreatedAt: r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			})
		}

		return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(gin.H{
			"folders": folders,
		})
	},
)
