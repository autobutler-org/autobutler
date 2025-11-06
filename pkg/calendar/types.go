package calendar

import (
	"strings"
	"time"
)

type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
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

type WeekMode int

const (
	WeekModeStandard WeekMode = iota // Week starts on Sunday
	WeekModeISO                      // Week starts on Monday
)

type CalendarView int

const (
	CalendarViewMonth CalendarView = iota
	CalendarViewWeek
	CalendarViewDay
)

type CalendarEvent struct {
	ID          int64
	Title       string
	Description string
	StartTime   time.Time
	EndTime     *time.Time
	AllDay      bool
	Location    string
	CalendarID  int64
}

type EventMap map[int][]*CalendarEvent

func NewCalendarEvent(
	title string,
	description string,
	startTime time.Time,
	allDay bool,
	location string,
	calendarId int64,
) *CalendarEvent {
	return &CalendarEvent{
		Title:       title,
		Description: description,
		StartTime:   startTime,
		EndTime:     nil,
		AllDay:      allDay,
		Location:    location,
		CalendarID:  calendarId,
	}
}

func NewCalendarEventWithEnd(
	title string,
	description string,
	startTime time.Time,
	endTime time.Time,
	allDay bool,
	location string,
	calendarId int64,
) *CalendarEvent {
	return &CalendarEvent{
		Title:       title,
		Description: description,
		StartTime:   startTime,
		EndTime:     &endTime,
		AllDay:      allDay,
		Location:    location,
		CalendarID:  calendarId,
	}
}

type MonthInfo struct {
	StartOfMonth  time.Time
	LeadingDays   int
	TrailingDays  int
	MonthDays     int
	TotalDays     int
	WeeksToRender int
}

func NewMonthInfo(now time.Time, totalDays int, totalDaysInMonth int, leadingEmptyDays int) MonthInfo {
	return MonthInfo{
		StartOfMonth:  time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
		LeadingDays:   leadingEmptyDays,
		TrailingDays:  totalDays - (leadingEmptyDays + totalDaysInMonth),
		MonthDays:     totalDaysInMonth,
		TotalDays:     totalDays,
		WeeksToRender: totalDays / 7,
	}
}

func NewMonthInfoFromTime(now time.Time) MonthInfo {
	firstOfMonth := GetFirstDayOfMonth(now)
	totalDaysInMonth := int(time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day())
	leadingEmptyDays := int(firstOfMonth.Weekday())
	totalDays := leadingEmptyDays + totalDaysInMonth
	if totalDays%7 != 0 {
		totalDays += 7 - (totalDays % 7) // Round up to the nearest week
	}
	return NewMonthInfo(now, totalDays, totalDaysInMonth, leadingEmptyDays)
}
