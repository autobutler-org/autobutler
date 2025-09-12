package calendar

import "time"

type EventMap map[int][]*Event

var eventStore EventMap

func init() {
	now := time.Now()
	events := []*Event{
		NewEvent(
			"Meeting with Bingus",
			"Discuss project updates",
			time.Date(now.Year(), now.Month(), 10, 14, 0, 0, 0, time.UTC),
			time.Date(now.Year(), now.Month(), 10, 14, 1, 0, 0, time.UTC),
		),
		NewEvent(
			"Meeting with Bingus's dumb cat",
			"Discuss project updates",
			time.Date(now.Year(), now.Month(), 10, 14, 1, 0, 0, time.UTC),
			time.Date(now.Year(), now.Month(), 10, 15, 2, 0, 0, time.UTC),
		),
		NewAllDayEvent(
			"Conference",
			"Annual tech conference",
			time.Date(now.Year(), now.Month(), 20, 0, 0, 0, 0, time.UTC),
		),
	}
	eventStore = make(EventMap, 0)
	for _, event := range events {
		day := event.StartTime.Day()
		if _, exists := eventStore[day]; !exists {
			eventStore[day] = []*Event{}
		}
		eventStore[day] = append(eventStore[day], event)
	}
}

func GetMonthEvents(_ time.Time) (EventMap, error) {
	// TOOD: Query sqlite instead
	return eventStore, nil
}
