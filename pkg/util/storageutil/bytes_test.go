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

func TestBytesToGB(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected float64
	}{
		{"zero", 0, 0},
		{"1 GB in bytes", 1073741824, 1.0},
		{"1 MB in bytes", 1048576, 1.0 / 1024},
		{"1 TB in bytes", 1099511627776, 1024.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BytesToGB(tt.input)
			if math.Abs(got-tt.expected) > 1e-9 {
				t.Errorf("BytesToGB(%d) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBytesToTB(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		expected float64
	}{
		{"zero", 0, 0},
		{"1 TB in bytes", 1099511627776, 1.0},
		{"1 GB in bytes", 1073741824, 1.0 / 1024},
		{"half TB", 549755813888, 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BytesToTB(tt.input)
			if math.Abs(got-tt.expected) > 1e-9 {
				t.Errorf("BytesToTB(%d) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestKBToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected uint64
	}{
		{"zero", 0, 0},
		{"1 KB", 1.0, 1024},
		{"0.5 KB", 0.5, 512},
		{"1024 KB = 1 MB", 1024.0, 1048576},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := KBToBytes(tt.input)
			if got != tt.expected {
				t.Errorf("KBToBytes(%v) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMBToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected uint64
	}{
		{"zero", 0, 0},
		{"1 MB", 1.0, 1048576},
		{"0.5 MB", 0.5, 524288},
		{"1024 MB = 1 GB", 1024.0, 1073741824},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MBToBytes(tt.input)
			if got != tt.expected {
				t.Errorf("MBToBytes(%v) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGBToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected uint64
	}{
		{"zero", 0, 0},
		{"1 GB", 1.0, 1073741824},
		{"0.5 GB", 0.5, 536870912},
		{"1024 GB = 1 TB", 1024.0, 1099511627776},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GBToBytes(tt.input)
			if got != tt.expected {
				t.Errorf("GBToBytes(%v) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTBToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected uint64
	}{
		{"zero", 0, 0},
		{"1 TB", 1.0, 1099511627776},
		{"0.5 TB", 0.5, 549755813888},
		{"2 TB", 2.0, 2199023255552},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TBToBytes(tt.input)
			if got != tt.expected {
				t.Errorf("TBToBytes(%v) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Verify that converting to a unit and back gives the original value
	original := uint64(1073741824) // 1 GB
	if KBToBytes(BytesToKB(original)) != original {
		t.Error("KB round-trip failed")
	}
	if MBToBytes(BytesToMB(original)) != original {
		t.Error("MB round-trip failed")
	}
	if GBToBytes(BytesToGB(original)) != original {
		t.Error("GB round-trip failed")
	}

	originalTB := uint64(1099511627776) // 1 TB
	if TBToBytes(BytesToTB(originalTB)) != originalTB {
		t.Error("TB round-trip failed")
	}
}
