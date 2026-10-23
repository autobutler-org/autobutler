package v0_events_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	v0_events "github.com/autobutler-org/quark/internal/server/api/v0/events"
	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/ctxutil"
	"github.com/autobutler-org/quark/pkg/util/deputil"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/serverutil"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/gin-gonic/gin"
)

func newEventsServer(t *testing.T, bus *eventbus.Bus) *httptest.Server {
	t.Helper()
	deps := deputil.NewDependencies().WithEventBus(bus)
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c = ctxutil.With(c, "deps", deps)
		c = ctxutil.With(c, "principal", accessutil.System)
		c.Next()
	})
	group := engine.Group("/api/v0")
	serverutil.RegisterRouterWithGroup(group, v0_events.NewRouter())
	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)
	return srv
}

func wsURL(srv *httptest.Server, path string) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") + path
}

// TestStreamEvents_ReceivesPublishedEvent verifies that an event published to
// the bus is delivered to a connected WebSocket client.
func TestStreamEvents_ReceivesPublishedEvent(t *testing.T) {
	bus := eventbus.New()
	srv := newEventsServer(t, bus)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL(srv, "/api/v0/events"), &websocket.DialOptions{})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.CloseNow()

	// Publish a test event.
	go bus.Publish(eventbus.Event{
		Kind: eventbus.EventUpload,
		Path: "photos/test.jpg",
	})

	// Read the event back.
	var evt eventbus.Event
	if err := wsjson.Read(ctx, conn, &evt); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if evt.Kind != eventbus.EventUpload {
		t.Errorf("Kind = %q; want %q", evt.Kind, eventbus.EventUpload)
	}
	if evt.Path != "photos/test.jpg" {
		t.Errorf("Path = %q; want 'photos/test.jpg'", evt.Path)
	}
}
