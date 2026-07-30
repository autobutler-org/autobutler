package photoutil

import (
	"encoding/binary"
	"encoding/hex"
	"image"
	"image/color"
	"math/bits"

	xdraw "golang.org/x/image/draw"
)

const (
	// dHashWidth is the number of columns in the dHash grid.
	// Each row produces (dHashWidth-1) bits of horizontal difference, so a
	// 9×8 grid yields 8×8 = 64 bits total.
	dHashWidth  = 9
	dHashHeight = 8
)

// DHash computes the difference hash (dHash) of img as a 64-bit integer.
//
// Algorithm:
//  1. Resize img to dHashWidth × dHashHeight (9 × 8) using bilinear interpolation.
//  2. Convert to greyscale.
//  3. For each row, compare adjacent pixel pairs left-to-right: set bit = 1
//     when left pixel is brighter than right pixel.
//
// The result is an 8-byte (64-bit) value. Two images with a Hamming distance
// <= 10 are considered near-duplicates in practice.
func DHash(img image.Image) uint64 {
	// Resize to 9x8 using bilinear interpolation.
	small := image.NewGray(image.Rect(0, 0, dHashWidth, dHashHeight))
	tmp := image.NewRGBA(small.Bounds())
	xdraw.BiLinear.Scale(tmp, tmp.Bounds(), img, img.Bounds(), xdraw.Over, nil)
	// Convert RGBA → grey.
	for y := 0; y < dHashHeight; y++ {
		for x := 0; x < dHashWidth; x++ {
			small.Set(x, y, color.GrayModel.Convert(tmp.At(x, y)))
		}
	}

	// Compute horizontal differences row by row.
	var hash uint64
	var bit uint64 = 1
	for y := 0; y < dHashHeight; y++ {
		for x := 0; x < dHashWidth-1; x++ {
			l := small.GrayAt(x, y).Y
			r := small.GrayAt(x+1, y).Y
			if l > r {
				hash |= bit
			}
			bit <<= 1
		}
	}
	return hash
}

// DHashHex returns the hex-encoded 16-character string representation of the
// dHash for img (suitable for storage in SQLite and indexed string comparison).
func DHashHex(img image.Image) string {
	h := DHash(img)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], h)
	return hex.EncodeToString(buf[:])
}

// HammingDistance returns the number of differing bits between two dHash
// values. Values <= 10 indicate near-duplicate images.
func HammingDistance(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// HammingDistanceHex parses two 16-char hex dHash strings and returns their
// Hamming distance. Returns -1 if either string is malformed.
func HammingDistanceHex(a, b string) int {
	da, err := hexToUint64(a)
	if err != nil {
		return -1
	}
	db, err := hexToUint64(b)
	if err != nil {
		return -1
	}
	return HammingDistance(da, db)
}

func hexToUint64(s string) (uint64, error) {
	raw, err := hex.DecodeString(s)
	if err != nil || len(raw) != 8 {
		return 0, err
	}
	return binary.BigEndian.Uint64(raw), nil
}
