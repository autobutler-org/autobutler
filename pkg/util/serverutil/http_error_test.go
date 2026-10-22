package serverutil

import (
	"net/http"
	"testing"
)

func TestHttpError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *HttpError
		wantMsg string
	}{
		{
			name:    "simple message",
			err:     &HttpError{StatusCode: 400, Message: "bad request"},
			wantMsg: "bad request",
		},
		{
			name:    "empty message",
			err:     &HttpError{StatusCode: 500, Message: ""},
			wantMsg: "",
		},
		{
			name:    "detailed message",
			err:     &HttpError{StatusCode: 404, Message: "resource not found: /api/users/42"},
			wantMsg: "resource not found: /api/users/42",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

func TestNewHttpError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
	}{
		{"bad request", http.StatusBadRequest, "invalid input"},
		{"not found", http.StatusNotFound, "not found"},
		{"internal error", http.StatusInternalServerError, "something broke"},
		{"unauthorized", http.StatusUnauthorized, "missing token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewHttpError(tt.statusCode, tt.message)
			if err.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", err.StatusCode, tt.statusCode)
			}
			if err.Message != tt.message {
				t.Errorf("Message = %q, want %q", err.Message, tt.message)
			}
			if err.Error() != tt.message {
				t.Errorf("Error() = %q, want %q", err.Error(), tt.message)
			}
		})
	}
}

func TestHttpError_ImplementsError(t *testing.T) {
	var _ error = (*HttpError)(nil)
}
