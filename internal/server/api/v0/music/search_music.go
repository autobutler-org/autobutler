package v0_music

import (
	"errors"
	"strings"

	"github.com/autobutler-org/quark/pkg/util/musicutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// searchMusic godoc
// @Summary Search the music library
// @Description Matches the query against title, artist, album artist, album and genre, best ranked first, and returns only tracks the caller can read. The snippet is tag text with the matched terms in <b>…</b>; escape it before rendering it as HTML. Answers 501 when the database is PostgreSQL (#2958).
// @Tags music
// @Produce json
// @Param q query string true "Search text; FTS5 column filters such as artist:nina pass through"
// @Param limit query int false "Most results: 50 when absent, at most 200"
// @Success 200 {object} SearchResponse
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Failure 501 {object} serverutil.Response "Not Implemented"
// @Security BearerAuth
// @Router /music/search [get]
func searchMusic(c *gin.Context) *serverutil.Response {
	query := c.Query("q")
	if strings.TrimSpace(query) == "" {
		return serverutil.BadRequest(errors.New("q is required"))
	}
	limit, err := queryInt(c, "limit")
	if err != nil {
		return serverutil.BadRequest(err)
	}
	who, refusal := loadCaller(c)
	if refusal != nil {
		return refusal
	}
	result, err := musicutil.Search(musicutil.SearchParams{
		Ctx:    c.Request.Context(),
		DB:     who.deps.Database().Db,
		Access: who.access,
		Query:  query,
		Limit:  limit,
	})
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return jsonOk(SearchResponse{Results: result.Results})
}

var searchMusicRoute = serverutil.ApiRoute("GET", "/music/search", searchMusic)
