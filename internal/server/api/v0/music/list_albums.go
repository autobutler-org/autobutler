package v0_music

import (
	"github.com/autobutler-org/quark/pkg/util/musicutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// listAlbums godoc
// @Summary List music albums
// @Description Returns a page of albums ordered by album artist and title, and how many there are. An album is listed when the caller can read one of its tracks, and its counts, year and cover come from those tracks alone. coverSerial and coverPath name the track to ask the thumbnail endpoint for.
// @Tags music
// @Produce json
// @Param artist query string false "Keep one album artist's albums, compared without case; empty names the untagged artist"
// @Param offset query int false "Albums to skip"
// @Param limit query int false "Page size: 50 when absent, at most 200"
// @Success 200 {object} AlbumsResponse
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /music/albums [get]
func listAlbums(c *gin.Context) *serverutil.Response {
	offset, limit, err := paging(c)
	if err != nil {
		return serverutil.BadRequest(err)
	}
	who, refusal := loadCaller(c)
	if refusal != nil {
		return refusal
	}
	result, err := musicutil.ListAlbums(musicutil.ListAlbumsParams{
		Ctx:    c.Request.Context(),
		DB:     who.deps.Database().Db,
		Access: who.access,
		Artist: optionalQuery(c, "artist"),
		Offset: offset,
		Limit:  limit,
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return jsonOk(AlbumsResponse{Albums: result.Albums, Total: result.Total})
}

var listAlbumsRoute = serverutil.ApiRoute("GET", "/music/albums", listAlbums)
