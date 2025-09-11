package calendar

import (
	"autobutler/pkg/calendar"
	"time"
)

type CalendarView int

const (
	CalendarViewMonth CalendarView = iota
	CalendarViewWeek
	CalendarViewDay
)

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
	firstOfMonth := calendar.GetFirstDayOfMonth(now)
	totalDaysInMonth := int(time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day())
	leadingEmptyDays := int(firstOfMonth.Weekday())
	totalDays := leadingEmptyDays + totalDaysInMonth
	if totalDays%7 != 0 {
		totalDays += 7 - (totalDays % 7) // Round up to the nearest week
	}
	return NewMonthInfo(now, totalDays, totalDaysInMonth, leadingEmptyDays)
}
