package v0_events

import (
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// nextID is a monotonically increasing counter used to assign unique subscriber
// IDs to each WebSocket connection, avoiding collisions from concurrent connects.
var nextID atomic.Uint64

// streamEvents godoc
// @Summary Stream real-time file/device events
// @Description Upgrades the connection to WebSocket and pushes JSON events for file system mutations (upload, delete, move, new_folder)
// @Tags events
// @Produce json
// @Success 101 {string} string "Switching Protocols"
// @Failure 400 {object} serverutil.Response "Bad Request"
// @Router /events [get]
func streamEvents(c *gin.Context) {
	deps, ok := ctxutil.Get[deputil.Dependencies](c, "deps")
	if !ok {
		c.Status(http.StatusInternalServerError)
		return
	}

	// InsecureSkipVerify disables nhooyr's built-in origin check.
	// The Flutter web app is served from a different origin than the API server,
	// so the default check (Origin == Host) always fails with 403.
	// Auth is already enforced via the requireAuth middleware (?token= / Bearer).
	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	defer conn.CloseNow()

	id := strconv.FormatUint(nextID.Add(1), 10)
	ch, unsub := deps.EventBus().Subscribe(id)
	defer unsub()

	ctx := conn.CloseRead(c.Request.Context())
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-ch:
			if !ok {
				return
			}
			if err := wsjson.Write(ctx, conn, evt); err != nil {
				return
			}
		}
	}
}

var streamEventsRoute = serverutil.NewRoute(
	"GET", "/events", streamEvents,
)
