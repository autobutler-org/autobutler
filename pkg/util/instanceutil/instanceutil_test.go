package instanceutil

import (
	"strings"
	"testing"
)

func TestGenerateID(t *testing.T) {
	id, err := GenerateID()
	if err != nil {
		t.Fatalf("GenerateID failed: %v", err)
	}

	// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Errorf("Expected 5 dash-separated parts, got %d: %q", len(parts), id)
	}
	if len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 ||
		len(parts[3]) != 4 || len(parts[4]) != 12 {
		t.Errorf("Unexpected UUID format: %q", id)
	}

	// Version bit: parts[2] should start with '4'
	if !strings.HasPrefix(parts[2], "4") {
		t.Errorf("Expected version 4 UUID, got: %q", parts[2])
	}
}

func TestGenerateID_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for range 100 {
		id, err := GenerateID()
		if err != nil {
			t.Fatalf("GenerateID failed: %v", err)
		}
		if seen[id] {
			t.Fatalf("Duplicate ID generated: %q", id)
		}
		seen[id] = true
	}
}
