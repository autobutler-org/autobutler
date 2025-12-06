package ctxutil

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWith(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := &gin.Context{}

	// Test setting a string value
	result := With(ctx, "key1", "value1")

	if result != ctx {
		t.Error("With should return the same context")
	}

	val, exists := ctx.Get("key1")
	if !exists {
		t.Error("Expected key1 to exist")
	}
	if val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}
}

func TestWith_MultipleTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := &gin.Context{}

	// Test various types
	With(ctx, "string", "hello")
	With(ctx, "int", 42)
	With(ctx, "bool", true)
	With(ctx, "struct", struct{ Name string }{"test"})

	tests := []struct {
		key      string
		expected interface{}
	}{
		{"string", "hello"},
		{"int", 42},
		{"bool", true},
		{"struct", struct{ Name string }{"test"}},
	}

	for _, tt := range tests {
		val, exists := ctx.Get(tt.key)
		if !exists {
			t.Errorf("Expected key %s to exist", tt.key)
		}
		if val != tt.expected {
			t.Errorf("For key %s, expected %v, got %v", tt.key, tt.expected, val)
		}
	}
}

func TestGet_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := &gin.Context{}
	ctx.Set("test_key", "test_value")

	value, exists := Get[string](ctx, "test_key")

	if !exists {
		t.Error("Expected key to exist")
	}
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}
}

func TestGet_NotExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := &gin.Context{}

	value, exists := Get[string](ctx, "nonexistent_key")

	if exists {
		t.Error("Expected key to not exist")
	}
	if value != "" {
		t.Errorf("Expected empty string, got '%s'", value)
	}
}

func TestGet_WrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := &gin.Context{}
	ctx.Set("test_key", 42) // Set as int

	// Try to get as string
	value, exists := Get[string](ctx, "test_key")

	if exists {
		t.Error("Expected exists to be false when type doesn't match")
	}
	if value != "" {
		t.Errorf("Expected empty string for wrong type, got '%s'", value)
	}
}

func TestGet_IntType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := &gin.Context{}
	ctx.Set("num", 123)

	value, exists := Get[int](ctx, "num")

	if !exists {
		t.Error("Expected key to exist")
	}
	if value != 123 {
		t.Errorf("Expected 123, got %d", value)
	}
}

func TestGet_StructType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	type TestStruct struct {
		Name string
		Age  int
	}

	ctx := &gin.Context{}
	expected := TestStruct{Name: "Alice", Age: 30}
	ctx.Set("person", expected)

	value, exists := Get[TestStruct](ctx, "person")

	if !exists {
		t.Error("Expected key to exist")
	}
	if value != expected {
		t.Errorf("Expected %+v, got %+v", expected, value)
	}
}

func TestGet_PointerType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := &gin.Context{}
	testValue := "pointer value"
	ctx.Set("ptr", &testValue)

	value, exists := Get[*string](ctx, "ptr")

	if !exists {
		t.Error("Expected key to exist")
	}
	if value == nil {
		t.Fatal("Expected non-nil pointer")
	}
	if *value != testValue {
		t.Errorf("Expected '%s', got '%s'", testValue, *value)
	}
}

func TestWith_ChainedCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := &gin.Context{}

	// Chain multiple With calls
	result := With(With(With(ctx, "a", 1), "b", 2), "c", 3)

	if result != ctx {
		t.Error("Chained With calls should return the same context")
	}

	// Verify all values were set
	if val, _ := Get[int](ctx, "a"); val != 1 {
		t.Errorf("Expected a=1, got %d", val)
	}
	if val, _ := Get[int](ctx, "b"); val != 2 {
		t.Errorf("Expected b=2, got %d", val)
	}
	if val, _ := Get[int](ctx, "c"); val != 3 {
		t.Errorf("Expected c=3, got %d", val)
	}
}
