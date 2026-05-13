package v1_thumbnails

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheKey(t *testing.T) {
	// Same inputs should produce the same key
	key1 := cacheKey("serial1", "/photos/test.jpg", 0, thumbnailSizeLg)
	key2 := cacheKey("serial1", "/photos/test.jpg", 0, thumbnailSizeLg)
	if key1 != key2 {
		t.Errorf("cacheKey not deterministic: %s != %s", key1, key2)
	}

	// Different serial should produce a different key
	key3 := cacheKey("serial2", "/photos/test.jpg", 0, thumbnailSizeLg)
	if key1 == key3 {
		t.Errorf("cacheKey collision on different serials: %s == %s", key1, key3)
	}

	// Different path should produce a different key
	key4 := cacheKey("serial1", "/photos/other.jpg", 0, thumbnailSizeLg)
	if key1 == key4 {
		t.Errorf("cacheKey collision on different paths: %s == %s", key1, key4)
	}

	// Empty serial should work (default case)
	key5 := cacheKey("", "/photos/test.jpg", 0, thumbnailSizeLg)
	if key5 == "" {
		t.Error("cacheKey returned empty string for empty serial")
	}
	if key5 == key1 {
		t.Errorf("cacheKey should differ for empty vs non-empty serial")
	}

	// Different rotation should produce a different key
	key6 := cacheKey("serial1", "/photos/test.jpg", 1, thumbnailSizeLg)
	if key1 == key6 {
		t.Errorf("cacheKey collision on different rotation: %s == %s", key1, key6)
	}

	// Different size should produce a different key
	keySm := cacheKey("serial1", "/photos/test.jpg", 0, thumbnailSizeSm)
	keyMd := cacheKey("serial1", "/photos/test.jpg", 0, thumbnailSizeMd)
	if key1 == keySm {
		t.Errorf("cacheKey collision: lg == sm")
	}
	if key1 == keyMd {
		t.Errorf("cacheKey collision: lg == md")
	}
	if keySm == keyMd {
		t.Errorf("cacheKey collision: sm == md")
	}
}

func TestParseThumbnailSize(t *testing.T) {
	tests := []struct {
		input    string
		expected thumbnailSize
	}{
		{"sm", thumbnailSizeSm},
		{"md", thumbnailSizeMd},
		{"lg", thumbnailSizeLg},
		{"", thumbnailSizeLg},   // empty defaults to lg
		{"xl", thumbnailSizeLg}, // unknown defaults to lg
	}
	for _, tt := range tests {
		got := parseThumbnailSize(tt.input)
		if got != tt.expected {
			t.Errorf("parseThumbnailSize(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestThumbnailDimensions(t *testing.T) {
	tests := []struct {
		size thumbnailSize
		w, h uint
	}{
		{thumbnailSizeSm, 96, 96},
		{thumbnailSizeMd, 240, 240},
		{thumbnailSizeLg, 400, 400},
		{"unknown", 400, 400}, // unknown falls back to lg
	}
	for _, tt := range tests {
		w, h := thumbnailDimensions(tt.size)
		if w != tt.w || h != tt.h {
			t.Errorf("thumbnailDimensions(%q) = (%d, %d), want (%d, %d)", tt.size, w, h, tt.w, tt.h)
		}
	}
}

func TestEtagFromModTime(t *testing.T) {
	now := time.Now()
	etag := etagFromModTime(now)

	// Should be quoted
	if etag[0] != '"' || etag[len(etag)-1] != '"' {
		t.Errorf("ETag should be quoted, got: %s", etag)
	}

	// Same time should produce same etag
	etag2 := etagFromModTime(now)
	if etag != etag2 {
		t.Errorf("etagFromModTime not deterministic: %s != %s", etag, etag2)
	}

	// Different time should produce different etag
	etag3 := etagFromModTime(now.Add(time.Second))
	if etag == etag3 {
		t.Errorf("etagFromModTime collision on different times: %s == %s", etag, etag3)
	}
}

func TestContentTypeForExt(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".webp", "image/jpeg"},
		{".gif", "image/jpeg"},
		{"", "image/jpeg"},
	}
	for _, tt := range tests {
		got := contentTypeForExt(tt.ext)
		if got != tt.expected {
			t.Errorf("contentTypeForExt(%q) = %q, want %q", tt.ext, got, tt.expected)
		}
	}
}

func TestThumbnailCacheDir(t *testing.T) {
	// thumbnailCacheDir should create the directory and return a valid path.
	// Skip if the data directory is not writable in this test environment.
	dir, err := thumbnailCacheDir()
	if err != nil {
		t.Skipf("skipping: cannot create cache dir in test environment: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("cache directory does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("cache path is not a directory: %s", dir)
	}

	// Should end with the expected path suffix
	if filepath.Base(dir) != "thumbnails" {
		t.Errorf("cache directory should end with 'thumbnails', got: %s", dir)
	}
}
