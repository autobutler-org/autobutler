package stringutil

import (
	"fmt"
	"strings"
)

// FormatNumber formats a number with commas for readability
func FormatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s,%03d", FormatNumber(n/1000), n%1000)
}

// TrimLeading trims leading characters from a string
func TrimLeading(s string, char rune) string {
	return strings.TrimLeft(s, string(char))
}
