package v0_calendar

import (
	"errors"

	"github.com/autobutler-org/quark/pkg/util/calendarutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
)

// moveEvent godoc
// @Summary Move a calendar event's occurrence
// @Description Moves one occurrence of a repeating event, that occurrence and every one after it, or the whole series. scope is one, following or all. occurrenceStart is the start of the occurrence that was moved, and start and end are where it was dropped, all RFC 3339. one records that start on the series and creates a one-off at the new time, owned by the series' owner. following ends the series the day before that occurrence and creates a new series at the new time, carrying later skipped starts shifted by the same amount; the first occurrence moves the whole series. all moves the series' own start and end as far as that occurrence moved. The series, and the created event when there is one, come back together.
// @Tags calendar
// @Accept json
// @Produce json
// @Param id path int true "Event ID"
// @Param body body moveRequest true "Which occurrence moved, and where"
// @Success 200 {object} MoveJSON
// @Failure 400 {object} serverutil.Response "Bad Request: a malformed id or body, or a move that breaks a rule"
// @Failure 404 {object} serverutil.Response "Not Found"
// @Failure 500 {object} serverutil.Response "Internal Server Error"
// @Security BearerAuth
// @Router /calendar/events/{id}/move [post]
func moveEvent(c *gin.Context) *serverutil.Response {
	id, ok := eventID(c)
	if !ok {
		return serverutil.BadRequest(errBadID)
	}
	var req moveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return serverutil.BadRequest(errors.New("the body must be JSON"))
	}
	occurrence, err := parseInstant("occurrenceStart", req.OccurrenceStart)
	if err != nil {
		return serverutil.BadRequest(err)
	}
	start, err := parseInstant("start", req.Start)
	if err != nil {
		return serverutil.BadRequest(err)
	}
	end, err := parseInstant("end", req.End)
	if err != nil {
		return serverutil.BadRequest(err)
	}

	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		return serverutil.InternalServerError(nil)
	}

	result, err := calendarutil.MoveEvent(c.Request.Context(), calendarutil.MoveEventParams{
		Database:        deps.Database(),
		ID:              id,
		Scope:           req.Scope,
		OccurrenceStart: occurrence,
		Start:           start,
		End:             end,
	})
	if err != nil {
		return eventError(err)
	}
	publishChanged(deps, id)
	body := MoveJSON{Series: toEventJSON(result.Series, callerID(c))}
	if result.Created != nil {
		created := toEventJSON(*result.Created, callerID(c))
		body.Created = &created
	}
	return serverutil.Ok().WithContentType(serverutil.ContentTypeJSON).WithData(body)
}

var moveEventRoute = serverutil.ApiRoute(
	"POST", "/calendar/events/:id/move", moveEvent,
)
