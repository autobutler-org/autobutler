package serverutil_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"autobutler/pkg/util/serverutil"

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
