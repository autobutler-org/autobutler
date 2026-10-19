package v0_music

import (
	"errors"

	"github.com/autobutler-org/quark/pkg/util/musicutil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// listTracks godoc
// @Summary List music tracks
// @Description Returns a sorted page of the tracks the caller can read, and how many there are. The default order is album order: album artist, album, disc, track; every sort breaks ties that way.
// @Tags music
// @Produce json
// @Param albumKey query string false "Keep one album's tracks"
// @Param artist query string false "Keep one album artist's tracks, compared without case; empty names the untagged artist"
// @Param genre query string false "Keep one genre's tracks, compared without case; empty names the untagged genre"
// @Param sort query string false "Sort key; album order when absent" Enums(title, artist, album, added, year)
// @Param order query string false "Sort direction" Enums(asc, desc)
// @Param offset query int false "Tracks to skip"
// @Param limit query int false "Page size: 50 when absent, at most 200"
// @Success 200 {object} TracksResponse
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /music/tracks [get]
func listTracks(c *gin.Context) *serverutil.Response {
	offset, limit, err := paging(c)
	if err != nil {
		return serverutil.BadRequest(err)
	}
	var descending bool
	switch c.Query("order") {
	case "", "asc":
	case "desc":
		descending = true
	default:
		return serverutil.BadRequest(errors.New("order must be asc or desc"))
	}
	who, refusal := loadCaller(c)
	if refusal != nil {
		return refusal
	}
	result, err := musicutil.ListTracks(musicutil.ListTracksParams{
		Ctx:        c.Request.Context(),
		DB:         who.deps.Database().Db,
		Access:     who.access,
		AlbumKey:   c.Query("albumKey"),
		Artist:     optionalQuery(c, "artist"),
		Genre:      optionalQuery(c, "genre"),
		Sort:       musicutil.TrackSort(c.Query("sort")),
		Descending: descending,
		Offset:     offset,
		Limit:      limit,
	})
	if errors.Is(err, musicutil.ErrInvalidSort) {
		return serverutil.BadRequest(err)
	}
	if err != nil {
		return serverutil.InternalServerError(err)
	}
	return jsonOk(TracksResponse{Tracks: result.Tracks, Total: result.Total})
}

var listTracksRoute = serverutil.ApiRoute("GET", "/music/tracks", listTracks)
