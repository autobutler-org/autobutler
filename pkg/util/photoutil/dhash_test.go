package photoutil_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/photoutil"
)

// solidImage returns a plain-colour image of the given size.
func solidImage(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// gradientImage returns an image with a right-to-left brightness gradient
// (left is bright, right is dark). After dHash resize this produces non-zero
// bits because each left pixel is brighter than its right neighbour.
func gradientImage(w, h int) image.Image {
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Bright on the left, dark on the right.
			v := uint8(255 - (x*255)/(w-1))
			img.SetGray(x, y, color.Gray{Y: v})
		}
	}
	return img
}

// TestDHash_SameImage verifies that hashing the same image twice yields the
// same hash (determinism).
func TestDHash_SameImage(t *testing.T) {
	img := gradientImage(128, 128)
	h1 := photoutil.DHash(img)
	h2 := photoutil.DHash(img)
	if h1 != h2 {
		t.Errorf("DHash not deterministic: %016x != %016x", h1, h2)
	}
}

// TestDHash_UniformImage verifies that a completely uniform (one solid colour)
// image hashes to 0 — no pixel differs from its neighbour.
func TestDHash_UniformImage(t *testing.T) {
	img := solidImage(64, 64, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	h := photoutil.DHash(img)
	if h != 0 {
		t.Errorf("uniform image: expected hash 0, got %016x", h)
	}
}

// TestDHash_DifferentImages verifies that visually different images (one dark,
// one light solid field, one gradient) produce different hashes.
func TestDHash_DifferentImages(t *testing.T) {
	dark := solidImage(64, 64, color.Black)
	light := solidImage(64, 64, color.White)
	grad := gradientImage(64, 64)

	hDark := photoutil.DHash(dark)
	hLight := photoutil.DHash(light)
	hGrad := photoutil.DHash(grad)

	// Dark and light are both uniform — both hash to 0.
	if hDark != 0 {
		t.Errorf("dark image: expected 0, got %016x", hDark)
	}
	if hLight != 0 {
		t.Errorf("light image: expected 0, got %016x", hLight)
	}
	// The gradient should differ from both.
	if hGrad == hDark {
		t.Error("gradient should differ from dark uniform image")
	}
}

// TestDHashHex verifies that DHashHex returns a 16-character hex string.
func TestDHashHex(t *testing.T) {
	img := gradientImage(64, 64)
	s := photoutil.DHashHex(img)
	if len(s) != 16 {
		t.Errorf("DHashHex length: got %d, want 16 (hex for 8 bytes)", len(s))
	}
	// Must be valid hex.
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("DHashHex contains non-hex character %q", c)
		}
	}
}

// TestHammingDistance verifies basic properties of Hamming distance.
func TestHammingDistance(t *testing.T) {
	// Identical hashes → distance 0.
	if d := photoutil.HammingDistance(0xDEADBEEF, 0xDEADBEEF); d != 0 {
		t.Errorf("identical hashes: expected 0, got %d", d)
	}
	// Single bit difference → distance 1.
	if d := photoutil.HammingDistance(0b0000, 0b0001); d != 1 {
		t.Errorf("single bit: expected 1, got %d", d)
	}
	// All bits differ → distance 64.
	if d := photoutil.HammingDistance(0, ^uint64(0)); d != 64 {
		t.Errorf("all bits: expected 64, got %d", d)
	}
}

// TestHammingDistanceHex_RoundTrip verifies that DHashHex + HammingDistanceHex
// is self-consistent: the same image hashes to distance 0.
func TestHammingDistanceHex_RoundTrip(t *testing.T) {
	img := gradientImage(128, 96)
	s := photoutil.DHashHex(img)
	d := photoutil.HammingDistanceHex(s, s)
	if d != 0 {
		t.Errorf("self-distance: expected 0, got %d", d)
	}
}

// TestHammingDistanceHex_Invalid verifies that malformed input returns -1.
func TestHammingDistanceHex_Invalid(t *testing.T) {
	if d := photoutil.HammingDistanceHex("notvalid", "alsonotvalid"); d != -1 {
		t.Errorf("invalid input: expected -1, got %d", d)
	}
}
