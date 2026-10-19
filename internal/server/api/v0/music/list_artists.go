package v0_music

import (
	"github.com/autobutler-org/quark/pkg/util/musicutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// listArtists godoc
// @Summary List music artists
// @Description Returns a page of album artists by name, and how many there are, counted over the tracks the caller can read.
// @Tags music
// @Produce json
// @Param offset query int false "Artists to skip"
// @Param limit query int false "Page size: 50 when absent, at most 200"
// @Success 200 {object} ArtistsResponse
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /music/artists [get]
func listArtists(c *gin.Context) *serverutil.Response {
	offset, limit, err := paging(c)
	if err != nil {
		return serverutil.BadRequest(err)
	}
	who, refusal := loadCaller(c)
	if refusal != nil {
		return refusal
	}
	result, err := musicutil.ListArtists(musicutil.ListArtistsParams{
		Ctx:    c.Request.Context(),
		DB:     who.deps.Database().Db,
		Access: who.access,
		Offset: offset,
		Limit:  limit,
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return jsonOk(ArtistsResponse{Artists: result.Artists, Total: result.Total})
}

var listArtistsRoute = serverutil.ApiRoute("GET", "/music/artists", listArtists)
