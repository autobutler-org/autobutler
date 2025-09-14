package calendar

import (
	"time"
)

var days = []string{
	"Sunday",
	"Monday",
	"Tuesday",
	"Wednesday",
	"Thursday",
	"Friday",
	"Saturday",
}

func MonthToInt(month time.Month) int {
	if month < time.January || month > time.December {
		return 0
	}
	return int(month)
}

func ShortMonth(month time.Month) string {
	if month < time.January || month > time.December {
		return ""
	}
	return month.String()[:3]
}

func WeekdayToString(day Weekday, mode WeekMode) string {
	if day < Sunday || day > Saturday {
		return ""
	}
	if mode == WeekModeISO {
		return days[(day+6)%7] // Shift so that Monday is 0
	}
	return days[day]
}

func WeekdayToShortString(day Weekday, mode WeekMode) string {
	return WeekdayToString(day, mode)[:3]
}

func GetFirstDayOfMonth(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}
