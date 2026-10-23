package ctxutil

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGet_WrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := &gin.Context{}
	ctx.Set("test_key", 42)

	value, exists := Get[string](ctx, "test_key")

	if exists {
		t.Error("Expected exists to be false when type doesn't match")
	}
	if value != "" {
		t.Errorf("Expected empty string for wrong type, got '%s'", value)
	}
}
