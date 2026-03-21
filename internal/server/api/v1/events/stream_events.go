package v1_events

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/ctxutil"
	"github.com/autobutler-org/autobutler/pkg/util/deputil"
	"github.com/autobutler-org/autobutler/pkg/util/serverutil"
	"github.com/gin-gonic/gin"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

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

	conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // origin check handled by CORS middleware
	})
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	defer conn.CloseNow()

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	ch, unsub := deps.EventBus().Subscribe(id)
	defer unsub()

	ctx := conn.CloseRead(context.Background())
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
