package photoutil_test

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/autobutler-org/quark/pkg/util/photoutil"
)

// --- RoundTo ---

func TestRoundTo_TwoDecimals(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{1.006, 1.01}, // 1.005 is not IEEE-754 representable — use 1.006
		{1.004, 1.0},
		{3.14159, 3.14},
		{0.0, 0.0},
		{-1.236, -1.24},
	}
	for _, tc := range cases {
		got := photoutil.RoundTo(tc.in, 2)
		if math.Abs(got-tc.want) > 1e-9 {
			t.Errorf("photoutil.RoundTo(%v, 2) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestRoundTo_ZeroDecimals(t *testing.T) {
	if got := photoutil.RoundTo(2.7, 0); got != 3.0 {
		t.Errorf("photoutil.RoundTo(2.7, 0) = %v, want 3.0", got)
	}
	if got := photoutil.RoundTo(2.3, 0); got != 2.0 {
		t.Errorf("photoutil.RoundTo(2.3, 0) = %v, want 2.0", got)
	}
}

func TestRoundTo_SixDecimals(t *testing.T) {
	got := photoutil.RoundTo(37.123456789, 6)
	want := 37.123457
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("RoundTo = %v, want %v", got, want)
	}
}

// --- SummarizeExif ---

func TestSummarizeExif_NilData_ReturnsNilFields(t *testing.T) {
	// nil ExifData: no fields set — should return nil.
	data := &photoutil.ExifData{}
	result := photoutil.SummarizeExif(data)
	if result != nil {
		t.Errorf("expected nil for empty ExifData, got %+v", result)
	}
}

func TestSummarizeExif_WithMake(t *testing.T) {
	data := &photoutil.ExifData{Make: "Apple"}
	result := photoutil.SummarizeExif(data)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Make == nil || *result.Make != "Apple" {
		t.Errorf("expected Make='Apple', got %v", result.Make)
	}
}

func TestSummarizeExif_WithModel(t *testing.T) {
	data := &photoutil.ExifData{Model: "iPhone 16 Pro"}
	result := photoutil.SummarizeExif(data)
	if result == nil || result.Model == nil || *result.Model != "iPhone 16 Pro" {
		t.Errorf("expected Model='iPhone 16 Pro'")
	}
}

func TestSummarizeExif_WithDateTaken(t *testing.T) {
	dt := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)
	data := &photoutil.ExifData{DateTaken: &dt}
	result := photoutil.SummarizeExif(data)
	if result == nil || result.DateTaken == nil {
		t.Fatal("expected DateTaken to be set")
	}
	if *result.DateTaken != "2024-06-15T12:30:00Z" {
		t.Errorf("unexpected DateTaken: %q", *result.DateTaken)
	}
}

func TestSummarizeExif_WithISO(t *testing.T) {
	iso := 800
	data := &photoutil.ExifData{ISO: iso}
	result := photoutil.SummarizeExif(data)
	if result == nil || result.ISO == nil || *result.ISO != iso {
		t.Errorf("expected ISO=800")
	}
}

func TestSummarizeExif_WithAperture(t *testing.T) {
	data := &photoutil.ExifData{Aperture: 1.8}
	result := photoutil.SummarizeExif(data)
	if result == nil || result.Aperture == nil {
		t.Fatal("expected Aperture to be set")
	}
	if math.Abs(*result.Aperture-1.8) > 1e-9 {
		t.Errorf("expected Aperture=1.8, got %v", *result.Aperture)
	}
}

func TestSummarizeExif_ShutterSpeed_Fraction(t *testing.T) {
	// 1/250
	data := &photoutil.ExifData{ShutterSpeed: [2]int64{1, 250}}
	result := photoutil.SummarizeExif(data)
	if result == nil || result.ShutterSpeed == nil {
		t.Fatal("expected ShutterSpeed")
	}
	if *result.ShutterSpeed != "1/250" {
		t.Errorf("expected '1/250', got %q", *result.ShutterSpeed)
	}
}

func TestSummarizeExif_ShutterSpeed_Reducible(t *testing.T) {
	// 1/500 stored as 2/1000 — should reduce to 1/500
	data := &photoutil.ExifData{ShutterSpeed: [2]int64{2, 1000}}
	result := photoutil.SummarizeExif(data)
	if result == nil || result.ShutterSpeed == nil {
		t.Fatal("expected ShutterSpeed")
	}
	// d%n == 0 → "1/%d" form: 1000/2 = 500
	if *result.ShutterSpeed != "1/500" {
		t.Errorf("expected '1/500', got %q", *result.ShutterSpeed)
	}
}

func TestSummarizeExif_ShutterSpeed_Zero(t *testing.T) {
	// numerator=0 → bulb/zero shutter
	data := &photoutil.ExifData{ShutterSpeed: [2]int64{0, 1}}
	result := photoutil.SummarizeExif(data)
	if result == nil || result.ShutterSpeed == nil {
		t.Fatal("expected ShutterSpeed")
	}
	if *result.ShutterSpeed != "0" {
		t.Errorf("expected '0', got %q", *result.ShutterSpeed)
	}
}

func TestSummarizeExif_WithGPS(t *testing.T) {
	data := &photoutil.ExifData{
		HasGPS:    true,
		Latitude:  37.7749295,
		Longitude: -122.4194155,
	}
	result := photoutil.SummarizeExif(data)
	if result == nil || result.Latitude == nil || result.Longitude == nil {
		t.Fatal("expected GPS fields")
	}
	if math.Abs(*result.Latitude-37.774930) > 1e-5 {
		t.Errorf("Latitude unexpected: %v", *result.Latitude)
	}
	if math.Abs(*result.Longitude-(-122.419416)) > 1e-5 {
		t.Errorf("Longitude unexpected: %v", *result.Longitude)
	}
}

func TestSummarizeExif_GPSFalse_NoCoords(t *testing.T) {
	data := &photoutil.ExifData{HasGPS: false, Latitude: 37.7, Longitude: -122.4}
	result := photoutil.SummarizeExif(data)
	// HasGPS=false means no GPS in the result (the coordinates are noise).
	if result != nil && result.Latitude != nil {
		t.Errorf("expected no GPS in result when HasGPS=false")
	}
}

// --- FindLivePhotoVideo ---

func TestFindLivePhotoVideo_NonImageExt(t *testing.T) {
	// Non-HEIC/JPG files: always returns ""
	result := photoutil.FindLivePhotoVideo("/tmp/doc.pdf", "doc.pdf")
	if result != "" {
		t.Errorf("expected '' for non-image file, got %q", result)
	}
}

func TestFindLivePhotoVideo_NoVideoSibling(t *testing.T) {
	tmp := t.TempDir()
	imgPath := filepath.Join(tmp, "photo.jpg")
	os.WriteFile(imgPath, []byte("fake"), 0600)
	// No corresponding .mov/.mp4 exists — should return ""
	result := photoutil.FindLivePhotoVideo(imgPath, "photo.jpg")
	if result != "" {
		t.Errorf("expected '' when no video sibling, got %q", result)
	}
}

func TestFindLivePhotoVideo_WithMovSibling(t *testing.T) {
	tmp := t.TempDir()
	imgPath := filepath.Join(tmp, "photo.jpg")
	movPath := filepath.Join(tmp, "photo.mov")
	os.WriteFile(imgPath, []byte("fake"), 0600)
	os.WriteFile(movPath, []byte("fake"), 0600)

	result := photoutil.FindLivePhotoVideo(imgPath, "photo.jpg")
	if result != "photo.mov" {
		t.Errorf("expected 'photo.mov', got %q", result)
	}
}

func TestFindLivePhotoVideo_HeicWithMov(t *testing.T) {
	tmp := t.TempDir()
	imgPath := filepath.Join(tmp, "burst.heic")
	movPath := filepath.Join(tmp, "burst.MOV")
	os.WriteFile(imgPath, []byte("fake"), 0600)
	os.WriteFile(movPath, []byte("fake"), 0600)

	result := photoutil.FindLivePhotoVideo(imgPath, "burst.heic")
	if result != "burst.MOV" {
		t.Errorf("expected 'burst.MOV', got %q", result)
	}
}

// The returned path must keep the album directory from relPath while using the
// video's real filename. Nothing covered a nested path before, and the name is
// handed to the client as livePhotoVideoPath.
func TestFindLivePhotoVideo_NestedRelPath(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "photo.jpg"), []byte("fake"), 0600)
	os.WriteFile(filepath.Join(tmp, "photo.mov"), []byte("fake"), 0600)

	result := photoutil.FindLivePhotoVideo(filepath.Join(tmp, "photo.jpg"), "albums/2024/photo.jpg")
	if result != "albums/2024/photo.mov" {
		t.Errorf("expected 'albums/2024/photo.mov', got %q", result)
	}
}

// A Live Photo's companion is a .mov, so it wins when a library holds both.
func TestFindLivePhotoVideo_PrefersMovOverMp4(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "photo.jpg"), []byte("fake"), 0600)
	os.WriteFile(filepath.Join(tmp, "photo.mp4"), []byte("fake"), 0600)
	os.WriteFile(filepath.Join(tmp, "photo.mov"), []byte("fake"), 0600)

	result := photoutil.FindLivePhotoVideo(filepath.Join(tmp, "photo.jpg"), "photo.jpg")
	if result != "photo.mov" {
		t.Errorf("expected 'photo.mov', got %q", result)
	}
}

// A video whose stem merely starts with the image's stem is a different file.
func TestFindLivePhotoVideo_IgnoresDifferentStem(t *testing.T) {
	tmp := t.TempDir()
	os.WriteFile(filepath.Join(tmp, "photo.jpg"), []byte("fake"), 0600)
	os.WriteFile(filepath.Join(tmp, "photo-2.mov"), []byte("fake"), 0600)

	result := photoutil.FindLivePhotoVideo(filepath.Join(tmp, "photo.jpg"), "photo.jpg")
	if result != "" {
		t.Errorf("expected '' for a non-matching stem, got %q", result)
	}
}
