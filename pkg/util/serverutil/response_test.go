package serverutil_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/serverutil"
)

func TestNewResponse_Defaults(t *testing.T) {
	resp := serverutil.NewResponse()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected StatusCode %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if resp.ContentType != serverutil.ContentTypeJSON {
		t.Errorf("expected ContentType %s, got %s", serverutil.ContentTypeJSON, resp.ContentType)
	}
	if resp.Data != nil {
		t.Errorf("expected Data nil, got %v", resp.Data)
	}
	if resp.Error != nil {
		t.Errorf("expected Error nil, got %v", resp.Error)
	}
}

func TestWithStatusCode(t *testing.T) {
	resp := serverutil.NewResponse()
	ret := resp.WithStatusCode(http.StatusCreated)

	if ret != resp {
		t.Error("WithStatusCode should return the same pointer")
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected StatusCode %d, got %d", http.StatusCreated, resp.StatusCode)
	}
}

func TestWithData(t *testing.T) {
	resp := serverutil.NewResponse()
	data := map[string]string{"key": "value"}
	ret := resp.WithData(data)

	if ret != resp {
		t.Error("WithData should return the same pointer")
	}
	if resp.Data == nil {
		t.Fatal("expected Data to be set")
	}
}

func TestWithError(t *testing.T) {
	resp := serverutil.NewResponse()
	err := fmt.Errorf("something broke")
	ret := resp.WithError(err)

	if ret != resp {
		t.Error("WithError should return the same pointer")
	}
	if resp.Error == nil || resp.Error.Error() != "something broke" {
		t.Errorf("expected error 'something broke', got %v", resp.Error)
	}
}

func TestWithContentType(t *testing.T) {
	resp := serverutil.NewResponse()
	ret := resp.WithContentType(serverutil.ContentTypeHTML)

	if ret != resp {
		t.Error("WithContentType should return the same pointer")
	}
	if resp.ContentType != serverutil.ContentTypeHTML {
		t.Errorf("expected ContentType %s, got %s", serverutil.ContentTypeHTML, resp.ContentType)
	}
}

func TestConvenienceConstructors(t *testing.T) {
	testErr := fmt.Errorf("test error")

	tests := []struct {
		name      string
		resp      *serverutil.Response
		wantCode  int
		wantError bool
	}{
		{"Ok", serverutil.Ok(), http.StatusOK, false},
		{"Accepted", serverutil.Accepted(), http.StatusAccepted, false},
		{"BadRequest", serverutil.BadRequest(testErr), http.StatusBadRequest, true},
		{"Unauthorized", serverutil.Unauthorized(testErr), http.StatusUnauthorized, true},
		{"NotFound", serverutil.NotFound(testErr), http.StatusNotFound, true},
		{"InternalServerError", serverutil.InternalServerError(testErr), http.StatusInternalServerError, true},
		{"ServiceUnavailable", serverutil.ServiceUnavailable(testErr), http.StatusServiceUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resp.StatusCode != tt.wantCode {
				t.Errorf("expected StatusCode %d, got %d", tt.wantCode, tt.resp.StatusCode)
			}
			if tt.resp.ContentType != serverutil.ContentTypeJSON {
				t.Errorf("expected ContentType %s, got %s", serverutil.ContentTypeJSON, tt.resp.ContentType)
			}
			if tt.wantError && tt.resp.Error == nil {
				t.Error("expected Error to be set")
			}
			if !tt.wantError && tt.resp.Error != nil {
				t.Errorf("expected nil Error, got %v", tt.resp.Error)
			}
			if tt.resp.Data != nil {
				t.Errorf("expected nil Data, got %v", tt.resp.Data)
			}
		})
	}
}

func TestChaining(t *testing.T) {
	resp := serverutil.Ok().WithData("hello").WithContentType(serverutil.ContentTypeHTML)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected StatusCode %d, got %d", http.StatusOK, resp.StatusCode)
	}
	if resp.Data != "hello" {
		t.Errorf("expected Data %q, got %v", "hello", resp.Data)
	}
	if resp.ContentType != serverutil.ContentTypeHTML {
		t.Errorf("expected ContentType %s, got %s", serverutil.ContentTypeHTML, resp.ContentType)
	}
	if resp.Error != nil {
		t.Errorf("expected nil Error, got %v", resp.Error)
	}
}
