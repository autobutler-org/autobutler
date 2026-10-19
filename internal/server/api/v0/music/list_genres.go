package v0_music

import (
	"github.com/autobutler-org/quark/pkg/util/musicutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// listGenres godoc
// @Summary List music genres
// @Description Returns every genre by name, counted over the tracks the caller can read.
// @Tags music
// @Produce json
// @Success 200 {object} GenresResponse
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /music/genres [get]
func listGenres(c *gin.Context) *serverutil.Response {
	who, refusal := loadCaller(c)
	if refusal != nil {
		return refusal
	}
	result, err := musicutil.ListGenres(musicutil.ListGenresParams{
		Ctx:    c.Request.Context(),
		DB:     who.deps.Database().Db,
		Access: who.access,
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return jsonOk(GenresResponse{Genres: result.Genres})
}

var listGenresRoute = serverutil.ApiRoute("GET", "/music/genres", listGenres)
