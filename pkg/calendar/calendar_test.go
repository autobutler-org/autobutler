package calendar

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// ParseMonth
// ---------------------------------------------------------------------------

func TestParseMonth(t *testing.T) {
	tests := []struct {
		input string
		want  Month
	}{
		// Full names
		{"January", January}, {"February", February}, {"March", March},
		{"April", April}, {"May", May}, {"June", June},
		{"July", July}, {"August", August}, {"September", September},
		{"October", October}, {"November", November}, {"December", December},

		// Short names
		{"Jan", January}, {"Feb", February}, {"Mar", March},
		{"Apr", April}, {"Jun", June}, {"Jul", July},
		{"Aug", August}, {"Sep", September}, {"Sept", September},
		{"Oct", October}, {"Nov", November}, {"Dec", December},

		// Numeric strings
		{"1", January}, {"2", February}, {"3", March},
		{"4", April}, {"5", May}, {"6", June},
		{"7", July}, {"8", August}, {"9", September},
		{"10", October}, {"11", November}, {"12", December},

		// Case insensitivity
		{"january", January}, {"JANUARY", January}, {"jAnUaRy", January},
		{"feb", February}, {"FEB", February},

		// Whitespace trimming
		{"  March  ", March},

		// Invalid
		{"", 0}, {"foo", 0}, {"13", 0}, {"0", 0}, {"-1", 0},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := ParseMonth(tc.input)
			if got != tc.want {
				t.Errorf("ParseMonth(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Month.IsValid
// ---------------------------------------------------------------------------

func TestMonthIsValid(t *testing.T) {
	tests := []struct {
		month Month
		want  bool
	}{
		{January, true}, {June, true}, {December, true},
		{0, false}, {13, false}, {-1, false},
	}

	for _, tc := range tests {
		got := tc.month.IsValid()
		if got != tc.want {
			t.Errorf("Month(%d).IsValid() = %v, want %v", tc.month, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Month.ToTimeMonth
// ---------------------------------------------------------------------------

func TestMonthToTimeMonth(t *testing.T) {
	tests := []struct {
		month Month
		want  time.Month
	}{
		{January, time.January},
		{June, time.June},
		{December, time.December},
	}

	for _, tc := range tests {
		got := tc.month.ToTimeMonth()
		if got != tc.want {
			t.Errorf("Month(%d).ToTimeMonth() = %v, want %v", tc.month, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// MonthToInt
// ---------------------------------------------------------------------------

func TestMonthToInt(t *testing.T) {
	tests := []struct {
		month time.Month
		want  int
	}{
		{time.January, 1}, {time.June, 6}, {time.December, 12},
		{time.Month(0), 0}, {time.Month(13), 0}, {time.Month(-1), 0},
	}

	for _, tc := range tests {
		got := MonthToInt(tc.month)
		if got != tc.want {
			t.Errorf("MonthToInt(%v) = %d, want %d", tc.month, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// ShortMonth
// ---------------------------------------------------------------------------

func TestShortMonth(t *testing.T) {
	expected := []string{
		"Jan", "Feb", "Mar", "Apr", "May", "Jun",
		"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
	}

	for i, want := range expected {
		month := time.Month(i + 1)
		got := ShortMonth(month)
		if got != want {
			t.Errorf("ShortMonth(%v) = %q, want %q", month, got, want)
		}
	}

	// Out-of-range
	if got := ShortMonth(time.Month(0)); got != "" {
		t.Errorf("ShortMonth(0) = %q, want \"\"", got)
	}
	if got := ShortMonth(time.Month(13)); got != "" {
		t.Errorf("ShortMonth(13) = %q, want \"\"", got)
	}
}

// ---------------------------------------------------------------------------
// WeekdayToString
// ---------------------------------------------------------------------------

func TestWeekdayToString(t *testing.T) {
	standardNames := []string{
		"Sunday", "Monday", "Tuesday", "Wednesday",
		"Thursday", "Friday", "Saturday",
	}

	// Standard mode: day value maps directly to day name.
	for i, want := range standardNames {
		got := WeekdayToString(Weekday(i), WeekModeStandard)
		if got != want {
			t.Errorf("WeekdayToString(%d, Standard) = %q, want %q", i, got, want)
		}
	}

	// ISO mode: shifts via (day+6)%7 index into the days slice.
	// The mapping treats the Weekday value as ISO-ordered (0=Monday).
	isoExpected := map[Weekday]string{
		Sunday:    "Saturday",
		Monday:    "Sunday",
		Tuesday:   "Monday",
		Wednesday: "Tuesday",
		Thursday:  "Wednesday",
		Friday:    "Thursday",
		Saturday:  "Friday",
	}
	for day, want := range isoExpected {
		got := WeekdayToString(day, WeekModeISO)
		if got != want {
			t.Errorf("WeekdayToString(%d, ISO) = %q, want %q", day, got, want)
		}
	}

	// Out-of-range
	if got := WeekdayToString(Weekday(-1), WeekModeStandard); got != "" {
		t.Errorf("WeekdayToString(-1, Standard) = %q, want \"\"", got)
	}
	if got := WeekdayToString(Weekday(7), WeekModeStandard); got != "" {
		t.Errorf("WeekdayToString(7, Standard) = %q, want \"\"", got)
	}
}

// ---------------------------------------------------------------------------
// WeekdayToShortString
// ---------------------------------------------------------------------------

func TestWeekdayToShortString(t *testing.T) {
	shortNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

	for i, want := range shortNames {
		got := WeekdayToShortString(Weekday(i), WeekModeStandard)
		if got != want {
			t.Errorf("WeekdayToShortString(%d, Standard) = %q, want %q", i, got, want)
		}
	}

	// ISO spot-check
	if got := WeekdayToShortString(Monday, WeekModeISO); got != "Sun" {
		t.Errorf("WeekdayToShortString(Monday, ISO) = %q, want \"Sun\"", got)
	}
}

// ---------------------------------------------------------------------------
// GetFirstDayOfMonth
// ---------------------------------------------------------------------------

func TestGetFirstDayOfMonth(t *testing.T) {
	tests := []struct {
		name  string
		input time.Time
	}{
		{"mid-month", time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)},
		{"first-day", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"last-day", time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)},
		{"leap-feb", time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetFirstDayOfMonth(tc.input)
			if got.Day() != 1 {
				t.Errorf("GetFirstDayOfMonth(%v).Day() = %d, want 1", tc.input, got.Day())
			}
			if got.Month() != tc.input.Month() {
				t.Errorf("GetFirstDayOfMonth(%v).Month() = %v, want %v", tc.input, got.Month(), tc.input.Month())
			}
			if got.Year() != tc.input.Year() {
				t.Errorf("GetFirstDayOfMonth(%v).Year() = %d, want %d", tc.input, got.Year(), tc.input.Year())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NewMonthInfoFromTime
// ---------------------------------------------------------------------------

func TestNewMonthInfoFromTime(t *testing.T) {
	tests := []struct {
		name          string
		input         time.Time
		leadingDays   int
		monthDays     int
		trailingDays  int
		totalDays     int
		weeksToRender int
	}{
		{
			// March 2026: starts on Sunday (weekday 0).
			// 31 days, 0 leading. 31%7=3 → totalDays=35, trailing=4, weeks=5.
			name:          "March 2026",
			input:         time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
			leadingDays:   0,
			monthDays:     31,
			trailingDays:  4,
			totalDays:     35,
			weeksToRender: 5,
		},
		{
			// February 2026: starts on Sunday (weekday 0).
			// 28 days, 0 leading. 28%7=0 → totalDays=28, trailing=0, weeks=4.
			name:          "February 2026",
			input:         time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
			leadingDays:   0,
			monthDays:     28,
			trailingDays:  0,
			totalDays:     28,
			weeksToRender: 4,
		},
		{
			// April 2026: starts on Wednesday (weekday 3).
			// 30 days, 3 leading. 33%7=5 → totalDays=35, trailing=2, weeks=5.
			name:          "April 2026",
			input:         time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
			leadingDays:   3,
			monthDays:     30,
			trailingDays:  2,
			totalDays:     35,
			weeksToRender: 5,
		},
		{
			// August 2026: starts on Saturday (weekday 6).
			// 31 days, 6 leading. 37%7=2 → totalDays=42, trailing=5, weeks=6.
			name:          "August 2026",
			input:         time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			leadingDays:   6,
			monthDays:     31,
			trailingDays:  5,
			totalDays:     42,
			weeksToRender: 6,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info := NewMonthInfoFromTime(tc.input)

			if info.LeadingDays != tc.leadingDays {
				t.Errorf("LeadingDays = %d, want %d", info.LeadingDays, tc.leadingDays)
			}
			if info.MonthDays != tc.monthDays {
				t.Errorf("MonthDays = %d, want %d", info.MonthDays, tc.monthDays)
			}
			if info.TrailingDays != tc.trailingDays {
				t.Errorf("TrailingDays = %d, want %d", info.TrailingDays, tc.trailingDays)
			}
			if info.TotalDays != tc.totalDays {
				t.Errorf("TotalDays = %d, want %d", info.TotalDays, tc.totalDays)
			}
			if info.WeeksToRender != tc.weeksToRender {
				t.Errorf("WeeksToRender = %d, want %d", info.WeeksToRender, tc.weeksToRender)
			}
			if info.StartOfMonth.Day() != 1 {
				t.Errorf("StartOfMonth.Day() = %d, want 1", info.StartOfMonth.Day())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NewCalendarEvent
// ---------------------------------------------------------------------------

func TestNewCalendarEvent(t *testing.T) {
	start := time.Date(2026, 3, 25, 14, 0, 0, 0, time.UTC)
	e := NewCalendarEvent("Meeting", "Team sync", start, false, "Office", 42)

	if e.Title != "Meeting" {
		t.Errorf("Title = %q, want %q", e.Title, "Meeting")
	}
	if e.Description != "Team sync" {
		t.Errorf("Description = %q, want %q", e.Description, "Team sync")
	}
	if !e.StartTime.Equal(start) {
		t.Errorf("StartTime = %v, want %v", e.StartTime, start)
	}
	if e.EndTime != nil {
		t.Errorf("EndTime = %v, want nil", e.EndTime)
	}
	if e.AllDay != false {
		t.Error("AllDay = true, want false")
	}
	if e.Location != "Office" {
		t.Errorf("Location = %q, want %q", e.Location, "Office")
	}
	if e.CalendarID != 42 {
		t.Errorf("CalendarID = %d, want 42", e.CalendarID)
	}
}

// ---------------------------------------------------------------------------
// NewCalendarEventWithEnd
// ---------------------------------------------------------------------------

func TestNewCalendarEventWithEnd(t *testing.T) {
	start := time.Date(2026, 3, 25, 14, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 25, 15, 0, 0, 0, time.UTC)
	e := NewCalendarEventWithEnd("Workshop", "Go testing", start, end, true, "Room B", 7)

	if e.Title != "Workshop" {
		t.Errorf("Title = %q, want %q", e.Title, "Workshop")
	}
	if e.EndTime == nil {
		t.Fatal("EndTime is nil, want non-nil")
	}
	if !e.EndTime.Equal(end) {
		t.Errorf("EndTime = %v, want %v", *e.EndTime, end)
	}
	if e.AllDay != true {
		t.Error("AllDay = false, want true")
	}
	if e.CalendarID != 7 {
		t.Errorf("CalendarID = %d, want 7", e.CalendarID)
	}
}
