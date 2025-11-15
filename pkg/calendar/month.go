package calendar

import (
	"strings"
	"time"
)

type Month int

const (
	January Month = iota + 1
	February
	March
	April
	May
	June
	July
	August
	September
	October
	November
	December
)

// ParseMonth converts a month string (name or number) to a Month value.
// Accepts full names (case-insensitive), short names, and numbers 1-12.
// Returns 0 if the input is invalid.
func ParseMonth(s string) Month {
	s = strings.ToLower(strings.TrimSpace(s))

	switch s {
	case "january", "jan", "1":
		return January
	case "february", "feb", "2":
		return February
	case "march", "mar", "3":
		return March
	case "april", "apr", "4":
		return April
	case "may", "5":
		return May
	case "june", "jun", "6":
		return June
	case "july", "jul", "7":
		return July
	case "august", "aug", "8":
		return August
	case "september", "sep", "sept", "9":
		return September
	case "october", "oct", "10":
		return October
	case "november", "nov", "11":
		return November
	case "december", "dec", "12":
		return December
	default:
		return 0
	}
}

// ToTimeMonth converts a Month to time.Month
func (m Month) ToTimeMonth() time.Month {
	return time.Month(m)
}

// IsValid returns true if the month is valid (1-12)
func (m Month) IsValid() bool {
	return m >= January && m <= December
}
