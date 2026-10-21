package calendarutil

import (
	"context"
	"time"

	"github.com/autobutler-org/quark/internal/db"
)

// moveEvent is MoveEvent without the transaction.
func moveEvent(ctx context.Context, q db.Querier, params MoveEventParams) (MoveEventResult, error) {
	if err := checkMove(params); err != nil {
		return MoveEventResult{}, err
	}
	got, err := GetEvent(ctx, GetEventParams{Queries: q, ID: params.ID})
	if err != nil {
		return MoveEventResult{}, err
	}
	event := got.Event
	if event.Repeat == RepeatNone && params.Scope != MoveAllEvents {
		return MoveEventResult{}, invalid("only a repeating event can move one occurrence")
	}
	loc := eventZone(event.TimeZone)
	// The first occurrence has no earlier part to leave behind, so moving it
	// and everything after it is the whole series.
	if params.Scope == MoveFollowing && !sameInstant(params.OccurrenceStart, event.Start) {
		return moveFollowing(ctx, q, event, params, loc)
	}
	if params.Scope == MoveThisEvent {
		return moveOne(ctx, q, event, params)
	}
	return moveAll(ctx, q, event, params, loc)
}

func checkMove(params MoveEventParams) error {
	switch params.Scope {
	case MoveThisEvent, MoveFollowing, MoveAllEvents:
	default:
		return invalid("scope must be one, following or all")
	}
	if params.OccurrenceStart.IsZero() || params.Start.IsZero() || params.End.IsZero() {
		return invalid("a start and an end are required")
	}
	if !params.End.After(params.Start) {
		return invalid("the event must end after it starts")
	}
	return nil
}

// moveOne skips the occurrence and stores its new time as a one-off event
// owned by the series' owner.
func moveOne(ctx context.Context, q db.Querier, event Event, params MoveEventParams) (MoveEventResult, error) {
	input := inputFrom(event)
	input.ExcludedStarts = append(append([]time.Time{}, event.ExcludedStarts...), params.OccurrenceStart)
	updated, err := UpdateEvent(ctx, UpdateEventParams{Queries: q, ID: event.ID, Input: input})
	if err != nil {
		return MoveEventResult{}, err
	}
	oneOff := inputFrom(event)
	oneOff.Start = params.Start
	oneOff.End = params.End
	oneOff.Repeat = RepeatNone
	oneOff.RepeatUntil = nil
	oneOff.ExcludedStarts = nil
	created, err := CreateEvent(ctx, CreateEventParams{Queries: q, Input: oneOff, CreatedBy: event.OwnerID})
	if err != nil {
		return MoveEventResult{}, err
	}
	return MoveEventResult{Series: updated.Event, Created: &created.Event}, nil
}

// moveFollowing ends the series the day before the occurrence and starts a
// new series at the dropped time. Exclusions on or after the occurrence move
// with it, shifted by the same amount.
func moveFollowing(ctx context.Context, q db.Querier, event Event, params MoveEventParams, loc *time.Location) (MoveEventResult, error) {
	before, after := splitExclusions(event.ExcludedStarts, params.OccurrenceStart)
	until := dayBefore(params.OccurrenceStart, loc)
	input := inputFrom(event)
	input.RepeatUntil = &until
	input.ExcludedStarts = before
	updated, err := UpdateEvent(ctx, UpdateEventParams{Queries: q, ID: event.ID, Input: input})
	if err != nil {
		return MoveEventResult{}, err
	}

	next := inputFrom(event)
	next.Start = params.Start
	next.End = params.End
	next.ExcludedStarts = shiftStarts(after, params.OccurrenceStart, params.Start, loc)
	if event.RepeatUntil != nil {
		kept := *event.RepeatUntil
		if startDate := calendarDate(params.Start, loc); startDate.After(kept) {
			kept = startDate
		}
		next.RepeatUntil = &kept
	}
	created, err := CreateEvent(ctx, CreateEventParams{Queries: q, Input: next, CreatedBy: event.OwnerID})
	if err != nil {
		return MoveEventResult{}, err
	}
	return MoveEventResult{Series: updated.Event, Created: &created.Event}, nil
}

// moveAll carries the series' own start and end as far as the occurrence
// moved, and shifts every exclusion with them.
func moveAll(ctx context.Context, q db.Querier, event Event, params MoveEventParams, loc *time.Location) (MoveEventResult, error) {
	input := inputFrom(event)
	input.Start = carried(event.Start, params.OccurrenceStart, params.Start, loc)
	input.End = carried(event.End, occurrenceEnd(event, params.OccurrenceStart), params.End, loc)
	input.ExcludedStarts = shiftStarts(event.ExcludedStarts, params.OccurrenceStart, params.Start, loc)
	updated, err := UpdateEvent(ctx, UpdateEventParams{Queries: q, ID: event.ID, Input: input})
	if err != nil {
		return MoveEventResult{}, err
	}
	return MoveEventResult{Series: updated.Event}, nil
}

// inputFrom copies the fields a move keeps. The caller replaces the ones
// that change.
func inputFrom(event Event) EventInput {
	return EventInput{
		Title:           event.Title,
		Notes:           event.Notes,
		Location:        event.Location,
		Start:           event.Start,
		End:             event.End,
		AllDay:          event.AllDay,
		TimeZone:        event.TimeZone,
		Repeat:          event.Repeat,
		RepeatUntil:     event.RepeatUntil,
		ReminderMinutes: event.ReminderMinutes,
		ColorIndex:      event.ColorIndex,
		ExcludedStarts:  event.ExcludedStarts,
	}
}

// carried is subject moved as far as from went to to: the same number of
// calendar dates, at to's time of day. Dates are counted in UTC numbers so a
// daylight saving day still counts as one.
func carried(subject, from, to time.Time, loc *time.Location) time.Time {
	origin := from.In(loc)
	landed := to.In(loc)
	base := subject.In(loc)
	originDate := time.Date(origin.Year(), origin.Month(), origin.Day(), 0, 0, 0, 0, time.UTC)
	landedDate := time.Date(landed.Year(), landed.Month(), landed.Day(), 0, 0, 0, 0, time.UTC)
	days := int(landedDate.Sub(originDate) / (24 * time.Hour))
	return time.Date(base.Year(), base.Month(), base.Day()+days, landed.Hour(), landed.Minute(), landed.Second(), 0, loc).UTC()
}

// occurrenceEnd is when the occurrence starting at start ends: the series'
// length later.
func occurrenceEnd(event Event, start time.Time) time.Time {
	if event.AllDay {
		return start.AddDate(0, 0, int(event.End.Sub(event.Start)/(24*time.Hour)))
	}
	return start.Add(event.End.Sub(event.Start))
}

// dayBefore is the calendar date before occurrence in loc, midnight UTC
// standing for that date, which is how a series' last date is stored.
func dayBefore(occurrence time.Time, loc *time.Location) time.Time {
	prev := occurrence.In(loc).AddDate(0, 0, -1)
	return time.Date(prev.Year(), prev.Month(), prev.Day(), 0, 0, 0, 0, time.UTC)
}

// calendarDate is t's year, month and day in loc, stored at midnight UTC.
func calendarDate(t time.Time, loc *time.Location) time.Time {
	year, month, day := t.In(loc).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// splitExclusions divides starts into those before occurrence and those after
// it. The occurrence itself is the one being moved, so it is in neither.
func splitExclusions(starts []time.Time, occurrence time.Time) (before, after []time.Time) {
	at := occurrence.UTC().Truncate(time.Second)
	for _, start := range starts {
		utc := start.UTC().Truncate(time.Second)
		switch {
		case utc.Before(at):
			before = append(before, utc)
		case utc.After(at):
			after = append(after, utc)
		}
	}
	return before, after
}

// shiftStarts moves each start as far as from went to to.
func shiftStarts(starts []time.Time, from, to time.Time, loc *time.Location) []time.Time {
	if len(starts) == 0 {
		return nil
	}
	out := make([]time.Time, len(starts))
	for i, start := range starts {
		out[i] = carried(start, from, to, loc)
	}
	return out
}

func sameInstant(a, b time.Time) bool {
	return a.UTC().Truncate(time.Second).Equal(b.UTC().Truncate(time.Second))
}

// isExcluded reports whether start is one of the series' skipped starts.
func isExcluded(starts []time.Time, start time.Time) bool {
	at := start.UTC().Truncate(time.Second)
	for _, skipped := range starts {
		if skipped.UTC().Truncate(time.Second).Equal(at) {
			return true
		}
	}
	return false
}
