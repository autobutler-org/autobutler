package v0_photos

import "math"

func roundTo(v float64, decimals int) float64 {
	factor := math.Pow(10, float64(decimals))
	return math.Round(v*factor) / factor
}
