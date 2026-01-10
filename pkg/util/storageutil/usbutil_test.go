package storageutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileTrim(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"trims whitespace", "  hello world \n", "hello world"},
		{"empty file", "", ""},
		{"no trim needed", "foo", "foo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "testfile")
			if err := os.WriteFile(file, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}
			got := readFileTrim(file)
			if got != tt.want {
				t.Errorf("readFileTrim() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsStorageDevice(t *testing.T) {
	tests := []struct {
		name         string
		product      string
		manufacturer string
		want         bool
	}{
		{"host controller", "xHCI Host Controller", "Linux", false},
		{"non-storage device", "Some Product", "Some Manufacturer", false},
		// Add more cases as needed
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, "product"), []byte(tt.product), 0644)
			os.WriteFile(filepath.Join(dir, "manufacturer"), []byte(tt.manufacturer), 0644)
			dev := &usbDevice{Path: dir}
			got := dev.IsStorageDevice()
			if got != tt.want {
				t.Errorf("IsStorageDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}
