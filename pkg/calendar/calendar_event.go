package calendar

import "time"

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
