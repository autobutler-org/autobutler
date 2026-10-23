package storageutil

import (
	"math"
	"testing"
)

func TestBytesToKB(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected float64
	}{
		{"zero", 0, 0},
		{"1024 bytes = 1 KB", 1024, 1.0},
		{"1 byte", 1, 1.0 / 1024},
		{"1048576 bytes = 1024 KB", 1048576, 1024.0},
		{"large value", 1099511627776, 1073741824.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BytesToKB(tt.input)
			if math.Abs(got-tt.expected) > 1e-9 {
				t.Errorf("BytesToKB(%d) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBytesToMB(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected float64
	}{
		{"zero", 0, 0},
		{"1048576 bytes = 1 MB", 1048576, 1.0},
		{"1024 bytes", 1024, 1.0 / 1024},
		{"1 GB in bytes", 1073741824, 1024.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BytesToMB(tt.input)
			if math.Abs(got-tt.expected) > 1e-9 {
				t.Errorf("BytesToMB(%d) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
