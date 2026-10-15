package odsutil

import (
	"encoding/xml"
	"math"
	"strconv"
	"strings"
	"time"
)

// cellValue is what a cell displays. The typed attribute wins when it
// parses; the cell's text is the fallback. Whitespace alone is an empty
// cell, so a styled-but-empty cell does not extend the sheet.
func cellValue(start xml.StartElement, text string) (string, error) {
	v := displayValue(start, text)
	if strings.TrimSpace(v) == "" {
		return "", nil
	}
	if len(v) > maxCellBytes {
		return "", tooLongCell()
	}
	return v, nil
}

func displayValue(start xml.StartElement, text string) string {
	shown := text
	if strings.TrimSpace(shown) == "" {
		shown = ""
	}
	switch attrOffice(start, "value-type") {
	case "boolean":
		return typedOrText(attrOffice(start, "boolean-value"), shown, formatBool)
	case "date":
		return typedOrText(attrOffice(start, "date-value"), shown, formatDate)
	case "time":
		return typedOrText(attrOffice(start, "time-value"), shown, formatDuration)
	case "float", "percentage", "currency":
		if v := strings.TrimSpace(attrOffice(start, "value")); v != "" {
			return v
		}
		return shown
	case "string":
		if shown != "" {
			return shown
		}
		return keep(attrOffice(start, "string-value"))
	default:
		if shown != "" {
			return shown
		}
		if v := strings.TrimSpace(attrOffice(start, "value")); v != "" {
			return v
		}
		return keep(attrOffice(start, "string-value"))
	}
}

// typedOrText uses the attribute when it is present and parses, and the
// cell text otherwise. An attribute that does not parse still wins over
// nothing, so a value is not dropped because its format was unexpected.
func typedOrText(raw, shown string, format func(string) (string, bool)) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return shown
	}
	if s, ok := format(raw); ok {
		return s
	}
	if shown != "" {
		return shown
	}
	return raw
}

// keep returns s when it holds something other than whitespace.
func keep(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}
	return s
}

func formatBool(v string) (string, bool) {
	switch strings.ToLower(v) {
	case "true", "1":
		return "TRUE", true
	case "false", "0":
		return "FALSE", true
	default:
		return "", false
	}
}

// dateLayouts are the W3C dates a spreadsheet writes, most specific first.
// A zone is kept as the clock time the file stored, not shifted to UTC.
var dateLayouts = []string{
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func formatDate(raw string) (string, bool) {
	var (
		parsed time.Time
		ok     bool
	)
	for _, layout := range dateLayouts {
		t, err := time.Parse(layout, raw)
		if err != nil {
			continue
		}
		parsed = t
		ok = true
		break
	}
	if !ok {
		return "", false
	}
	if !strings.Contains(raw, "T") {
		return parsed.Format("2006-01-02"), true
	}
	return parsed.Round(time.Second).Format("2006-01-02 15:04:05"), true
}

// formatDuration reads an xsd:duration and writes it as hours:minutes:seconds.
// Hours are not wrapped at 24, so a duration of a day and an hour stays
// 25:00:00. Years and months have no fixed length and are left unparsed.
func formatDuration(raw string) (string, bool) {
	s := raw
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	if !strings.HasPrefix(s, "P") || len(s) < 2 {
		return "", false
	}
	s = s[1:]

	var (
		days, hours, mins int64
		secs              float64
		inTime            bool
		saw               bool
	)
	for len(s) > 0 {
		if s[0] == 'T' {
			if inTime {
				return "", false
			}
			inTime = true
			s = s[1:]
			continue
		}
		i := 0
		for i < len(s) && (s[i] == '.' || (s[i] >= '0' && s[i] <= '9')) {
			i++
		}
		if i == 0 || i == len(s) {
			return "", false
		}
		num, unit := s[:i], s[i]
		s = s[i+1:]
		saw = true
		switch unit {
		case 'Y':
			return "", false
		case 'M':
			if !inTime || strings.Contains(num, ".") {
				return "", false
			}
			n, ok := parseIntPart(num)
			if !ok {
				return "", false
			}
			mins = n
		case 'D':
			if inTime {
				return "", false
			}
			n, ok := parseIntPart(num)
			if !ok {
				return "", false
			}
			days = n
		case 'H':
			if !inTime {
				return "", false
			}
			n, ok := parseIntPart(num)
			if !ok {
				return "", false
			}
			hours = n
		case 'S':
			if !inTime {
				return "", false
			}
			f, err := strconv.ParseFloat(num, 64)
			if err != nil || f < 0 || math.IsInf(f, 0) || math.IsNaN(f) {
				return "", false
			}
			secs = f
		default:
			return "", false
		}
	}
	if !saw {
		return "", false
	}
	// A component that does not fit in a clock is left unparsed, so the
	// caller can fall back to the cell text instead of wrapping.
	total, ok := mulInt(days, 86400)
	if !ok {
		return "", false
	}
	more, ok := mulInt(hours, 3600)
	if !ok {
		return "", false
	}
	total, ok = addInt(total, more)
	if !ok {
		return "", false
	}
	more, ok = mulInt(mins, 60)
	if !ok {
		return "", false
	}
	total, ok = addInt(total, more)
	if !ok {
		return "", false
	}
	if math.IsInf(secs, 0) || secs >= float64(math.MaxInt64) {
		return "", false
	}
	total, ok = addInt(total, int64(math.Round(secs)))
	if !ok {
		return "", false
	}
	sign := ""
	if neg && total != 0 {
		sign = "-"
	}
	h := total / 3600
	m := (total % 3600) / 60
	sec := total % 60
	return sign + fmtClock(h, m, sec), true
}

func parseIntPart(num string) (int64, bool) {
	if num == "" || strings.Contains(num, ".") {
		return 0, false
	}
	n, err := strconv.ParseInt(num, 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func fmtClock(h, m, s int64) string {
	return pad(h) + ":" + pad(m) + ":" + pad(s)
}

func pad(n int64) string {
	if n < 10 {
		return "0" + strconv.FormatInt(n, 10)
	}
	return strconv.FormatInt(n, 10)
}
