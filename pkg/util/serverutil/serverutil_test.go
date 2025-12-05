package serverutil_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"autobutler/pkg/util/serverutil"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

// writeResponse is a helper that writes an serverutil.Response to a recorder
// mimicking the behavior of wrapApiRoute in serverutil
func writeResponse(resp *serverutil.Response) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	switch resp.ContentType {
	case serverutil.ContentTypeJSON:
		c.JSON(resp.StatusCode, resp.Data)
	case serverutil.ContentTypeHTML:
		c.String(resp.StatusCode, "%v", resp.Data)
	}

	return w
}

func TestNewResponse_Success(t *testing.T) {
	resp := serverutil.NewResponse().
		WithContentType(serverutil.ContentTypeJSON).
		WithStatusCode(http.StatusOK).
		WithData(map[string]string{"message": "success"})

	w := writeResponse(resp)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json; charset=utf-8, got %s", contentType)
	}

	expected := `{"message":"success"}`
	if w.Body.String() != expected {
		t.Errorf("expected body %q, got %q", expected, w.Body.String())
	}
}

func TestNewResponse_Error(t *testing.T) {
	resp := serverutil.NewResponse().
		WithContentType(serverutil.ContentTypeJSON).
		WithStatusCode(http.StatusInternalServerError).
		WithData(map[string]string{"error": "server error"})

	w := writeResponse(resp)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("expected Content-Type application/json; charset=utf-8, got %s", contentType)
	}
}

func TestNewResponse_HTML(t *testing.T) {
	resp := serverutil.NewResponse().
		WithContentType(serverutil.ContentTypeHTML).
		WithStatusCode(http.StatusBadRequest).
		WithData("error message")

	w := writeResponse(resp)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("expected Content-Type text/plain; charset=utf-8, got %s", contentType)
	}

	expected := "error message"
	if w.Body.String() != expected {
		t.Errorf("expected body %q, got %q", expected, w.Body.String())
	}
}

func TestOk(t *testing.T) {
	resp := serverutil.Ok()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	if resp.ContentType != serverutil.ContentTypeHTML {
		t.Errorf("expected content type %s, got %s", serverutil.ContentTypeHTML, resp.ContentType)
	}

	if resp.Data != nil {
		t.Errorf("expected nil data, got %v", resp.Data)
	}

	if resp.Error != nil {
		t.Errorf("expected nil error, got %v", resp.Error)
	}
}

func TestWithComponent(t *testing.T) {
	// Create a simple mock component
	mockComponent := &mockTemplComponent{content: "<div>Hello World</div>"}

	resp := serverutil.NewResponse().WithComponent(mockComponent)

	if resp.Data == nil {
		t.Error("expected Data to be set after WithComponent")
	}

	dataStr, ok := resp.Data.(string)
	if !ok {
		t.Errorf("expected Data to be a string, got %T", resp.Data)
	}

	if dataStr != "<div>Hello World</div>" {
		t.Errorf("expected Data to be '<div>Hello World</div>', got '%s'", dataStr)
	}
}

// mockTemplComponent is a mock implementation of templ.Component for testing
type mockTemplComponent struct {
	content string
}

func (m *mockTemplComponent) Render(ctx context.Context, w io.Writer) error {
	_, err := w.Write([]byte(m.content))
	return err
}

// mockFailingComponent is a mock component that always fails to render
type mockFailingComponent struct{}

func (m *mockFailingComponent) Render(ctx context.Context, w io.Writer) error {
	return fmt.Errorf("render failed")
}

func TestNewRoute(t *testing.T) {
	handler := func(c *gin.Context) {
		c.String(200, "OK")
	}

	route := serverutil.NewRoute("GET", "/test", handler)

	if route.Method != "GET" {
		t.Errorf("expected method GET, got %s", route.Method)
	}

	if route.Path != "/test" {
		t.Errorf("expected path /test, got %s", route.Path)
	}

	if route.Handler == nil {
		t.Error("expected handler to be non-nil")
	}
}

func TestApiRoute(t *testing.T) {
	handler := func(c *gin.Context) *serverutil.Response {
		return serverutil.NewResponse().
			WithStatusCode(http.StatusOK).
			WithContentType(serverutil.ContentTypeJSON).
			WithData(map[string]string{"status": "ok"})
	}

	route := serverutil.ApiRoute("POST", "/api/test", handler)

	if route.Method != "POST" {
		t.Errorf("expected method POST, got %s", route.Method)
	}

	if route.Path != "/api/test" {
		t.Errorf("expected path /api/test, got %s", route.Path)
	}

	if route.Handler == nil {
		t.Error("expected handler to be non-nil")
	}
}

func TestWrapApiRoute_WithData(t *testing.T) {
	handler := func(c *gin.Context) *serverutil.Response {
		return serverutil.NewResponse().
			WithStatusCode(http.StatusOK).
			WithContentType(serverutil.ContentTypeJSON).
			WithData(map[string]string{"message": "success"})
	}

	wrapped := serverutil.WrapApiRoute(handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	wrapped(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestWrapApiRoute_WithError(t *testing.T) {
	handler := func(c *gin.Context) *serverutil.Response {
		return serverutil.NewResponse().
			WithStatusCode(http.StatusInternalServerError).
			WithContentType(serverutil.ContentTypeJSON).
			WithError(http.ErrBodyNotAllowed)
	}

	wrapped := serverutil.WrapApiRoute(handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	wrapped(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestWrapApiRoute_StatusCodeOnly(t *testing.T) {
	// Test when both Data and Error are nil
	handler := func(c *gin.Context) *serverutil.Response {
		return serverutil.NewResponse().WithStatusCode(http.StatusNoContent)
	}

	wrapped := serverutil.WrapApiRoute(handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	wrapped(c)

	// Gin defaults to 200 when c.Status() is called without writing body
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestWrapApiRoute_HTMLError(t *testing.T) {
	handler := func(c *gin.Context) *serverutil.Response {
		return serverutil.NewResponse().
			WithStatusCode(http.StatusBadRequest).
			WithContentType(serverutil.ContentTypeHTML).
			WithError(fmt.Errorf("bad request"))
	}

	wrapped := serverutil.WrapApiRoute(handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	wrapped(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !strings.Contains(w.Body.String(), "bad request") {
		t.Errorf("expected body to contain 'bad request', got: %s", w.Body.String())
	}
}

func TestWrapApiRoute_HTMLData(t *testing.T) {
	handler := func(c *gin.Context) *serverutil.Response {
		return serverutil.NewResponse().
			WithStatusCode(http.StatusOK).
			WithContentType(serverutil.ContentTypeHTML).
			WithData("Hello World")
	}

	wrapped := serverutil.WrapApiRoute(handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	wrapped(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Hello World") {
		t.Errorf("expected body to contain 'Hello World', got: %s", w.Body.String())
	}
}

func TestWrapApiRoute_JSONError(t *testing.T) {
	handler := func(c *gin.Context) *serverutil.Response {
		return serverutil.NewResponse().
			WithStatusCode(http.StatusNotFound).
			WithContentType(serverutil.ContentTypeJSON).
			WithError(fmt.Errorf("not found"))
	}

	wrapped := serverutil.WrapApiRoute(handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	wrapped(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	if !strings.Contains(w.Body.String(), "not found") {
		t.Errorf("expected body to contain 'not found', got: %s", w.Body.String())
	}
}

func TestWrapApiRoute_UnsupportedContentType(t *testing.T) {
	handler := func(c *gin.Context) *serverutil.Response {
		resp := serverutil.NewResponse().
			WithStatusCode(http.StatusOK).
			WithData("test")
		// Manually set an unsupported ContentType
		resp.ContentType = "application/xml"
		return resp
	}

	wrapped := serverutil.WrapApiRoute(handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	wrapped(c)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	if !strings.Contains(w.Body.String(), "Unsupported content type") {
		t.Errorf("expected body to contain 'Unsupported content type', got: %s", w.Body.String())
	}
}

func TestWrapUiRoute_NilComponent(t *testing.T) {
	handler := func(c *gin.Context) templ.Component {
		return nil
	}

	wrapped := serverutil.WrapUiRoute(handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	wrapped(c)

	// The wrapped handler returns response with StatusCode 400 but no data/error,
	// so Gin defaults to 200
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestWrapUiRoute_RenderError(t *testing.T) {
	// Create a component that fails to render
	handler := func(c *gin.Context) templ.Component {
		return &mockFailingComponent{}
	}

	wrapped := serverutil.WrapUiRoute(handler)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// Need to set up a request with a valid context
	c.Request = httptest.NewRequest("GET", "/", nil)

	wrapped(c)

	// The wrapped handler returns response with StatusCode 400 but no data/error,
	// so Gin defaults to 200
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

type mockRouter struct {
	routes []*serverutil.Route
}

func (m *mockRouter) Routes() []*serverutil.Route {
	return m.routes
}

func TestRegisterRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	router := &mockRouter{
		routes: []*serverutil.Route{
			serverutil.NewRoute("GET", "/test", func(c *gin.Context) {
				c.String(200, "OK")
			}),
			serverutil.NewRoute("POST", "/test", func(c *gin.Context) {
				c.String(201, "Created")
			}),
		},
	}

	serverutil.RegisterRouter(engine, router)

	// Test GET route
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected status 200 for GET /test, got %d", w.Code)
	}

	// Test POST route
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/test", nil)
	engine.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("expected status 201 for POST /test, got %d", w.Code)
	}
}

func TestRegisterRouterWithGroup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("/api")

	router := &mockRouter{
		routes: []*serverutil.Route{
			serverutil.NewRoute("GET", "/users", func(c *gin.Context) {
				c.String(200, "users")
			}),
		},
	}

	serverutil.RegisterRouterWithGroup(group, router)

	// Test route within group
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users", nil)
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected status 200 for GET /api/users, got %d", w.Code)
	}
}
