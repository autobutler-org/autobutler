package calendarutil_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/calendarutil"
)

func weeklySeries(t *testing.T, q db.Querier, owner int64) calendarutil.Event {
	t.Helper()
	until := at("2026-10-31T00:00:00Z")
	created, err := calendarutil.CreateEvent(context.Background(), calendarutil.CreateEventParams{
		Queries: q,
		Input: calendarutil.EventInput{
			Title: "Practice", Notes: "Bring water.", Location: "Field",
			Start: at("2026-09-01T15:00:00Z"), End: at("2026-09-01T16:00:00Z"),
			TimeZone: "UTC", Repeat: calendarutil.RepeatWeekly, RepeatUntil: &until,
			ReminderMinutes: minutes(15), ColorIndex: 3,
			ExcludedStarts: []time.Time{at("2026-09-15T15:00:00Z"), at("2026-10-20T15:00:00Z")},
		},
		CreatedBy: owner,
	})
	if err != nil {
		t.Fatalf("CreateEvent: %v", err)
	}
	return created.Event
}

func moveSeries(t *testing.T, database *db.DatabaseSqlc, id int64, scope, occurrence, start, end string) calendarutil.MoveEventResult {
	t.Helper()
	result, err := calendarutil.MoveEvent(context.Background(), calendarutil.MoveEventParams{
		Database: database, ID: id, Scope: scope,
		OccurrenceStart: at(occurrence), Start: at(start), End: at(end),
	})
	if err != nil {
		t.Fatalf("MoveEvent(%s): %v", scope, err)
	}
	return result
}

func TestMoveThisEventSkipsTheOccurrence(t *testing.T) {
	database := dbtest.NewDB(t)
	owner := makeUser(t, database.Queries)
	series := weeklySeries(t, database.Queries, owner)

	moved := moveSeries(t, database, series.ID, calendarutil.MoveThisEvent,
		"2026-10-13T15:00:00Z", "2026-10-14T16:00:00Z", "2026-10-14T17:30:00Z")

	if moved.Created == nil {
		t.Fatal("moving one occurrence created no event")
	}
	one := moved.Created
	if one.Repeat != calendarutil.RepeatNone || one.OwnerID != owner || one.Title != "Practice" ||
		one.Notes != "Bring water." || one.Location != "Field" || one.ColorIndex != 3 ||
		one.ReminderMinutes == nil || *one.ReminderMinutes != 15 || one.TimeZone != "UTC" {
		t.Errorf("one-off = %+v, want the series' fields under its owner", one)
	}
	if !one.Start.Equal(at("2026-10-14T16:00:00Z")) || !one.End.Equal(at("2026-10-14T17:30:00Z")) {
		t.Errorf("one-off times = %s..%s", one.Start, one.End)
	}
	if !moved.Series.Start.Equal(series.Start) || moved.Series.RepeatUntil == nil ||
		!moved.Series.RepeatUntil.Equal(at("2026-10-31T00:00:00Z")) {
		t.Errorf("series = %+v, want it unchanged apart from the skipped start", moved.Series)
	}
	if !sameTimes(moved.Series.ExcludedStarts, []string{
		"2026-09-15T15:00:00Z", "2026-10-13T15:00:00Z", "2026-10-20T15:00:00Z",
	}) {
		t.Errorf("excluded = %v", moved.Series.ExcludedStarts)
	}

	starts, reminders := expand(t, moved.Series, "2026-10-01T00:00:00Z", "2026-11-01T00:00:00Z")
	equal(t, "series in October", starts, []string{
		"2026-10-06T15:00:00Z", "2026-10-27T15:00:00Z",
	})
	if reminders[0] != "2026-10-06T14:45:00Z" {
		t.Errorf("reminder = %s, want 15 minutes before the occurrence that remains", reminders[0])
	}
}

func TestMoveFollowingSplitsTheSeries(t *testing.T) {
	database := dbtest.NewDB(t)
	owner := makeUser(t, database.Queries)
	series := weeklySeries(t, database.Queries, owner)

	moved := moveSeries(t, database, series.ID, calendarutil.MoveFollowing,
		"2026-10-13T15:00:00Z", "2026-10-14T16:00:00Z", "2026-10-14T17:30:00Z")

	if moved.Series.RepeatUntil == nil || !moved.Series.RepeatUntil.Equal(at("2026-10-12T00:00:00Z")) {
		t.Errorf("original ends %v, want the day before the moved occurrence", moved.Series.RepeatUntil)
	}
	if !moved.Series.Start.Equal(series.Start) {
		t.Errorf("original start = %s, want it kept", moved.Series.Start)
	}
	if !sameTimes(moved.Series.ExcludedStarts, []string{"2026-09-15T15:00:00Z"}) {
		t.Errorf("original exclusions = %v, want only the one before the split", moved.Series.ExcludedStarts)
	}
	starts, _ := expand(t, moved.Series, "2026-09-01T00:00:00Z", "2026-11-01T00:00:00Z")
	equal(t, "original occurrences", starts, []string{
		"2026-09-01T15:00:00Z", "2026-09-08T15:00:00Z", "2026-09-22T15:00:00Z",
		"2026-09-29T15:00:00Z", "2026-10-06T15:00:00Z",
	})

	if moved.Created == nil {
		t.Fatal("the split created no series")
	}
	next := moved.Created
	if next.OwnerID != owner || next.Repeat != calendarutil.RepeatWeekly || next.ID == series.ID {
		t.Errorf("new series = %+v, want a weekly series under the same owner", next)
	}
	if !next.Start.Equal(at("2026-10-14T16:00:00Z")) || !next.End.Equal(at("2026-10-14T17:30:00Z")) {
		t.Errorf("new series times = %s..%s", next.Start, next.End)
	}
	if next.RepeatUntil == nil || !next.RepeatUntil.Equal(at("2026-10-31T00:00:00Z")) {
		t.Errorf("new series ends %v, want the original end date", next.RepeatUntil)
	}
	// Oct 20's exclusion moved a day and an hour with the occurrence.
	if !sameTimes(next.ExcludedStarts, []string{"2026-10-21T16:00:00Z"}) {
		t.Errorf("new exclusions = %v, want Oct 20 shifted to Oct 21 at 16:00", next.ExcludedStarts)
	}
	nextStarts, _ := expand(t, *next, "2026-10-01T00:00:00Z", "2026-11-08T00:00:00Z")
	equal(t, "new occurrences", nextStarts, []string{"2026-10-14T16:00:00Z", "2026-10-28T16:00:00Z"})
}

func TestMoveFollowingPastTheEndDateExtendsIt(t *testing.T) {
	database := dbtest.NewDB(t)
	series := weeklySeries(t, database.Queries, 0)

	moved := moveSeries(t, database, series.ID, calendarutil.MoveFollowing,
		"2026-10-27T15:00:00Z", "2026-11-03T15:00:00Z", "2026-11-03T16:00:00Z")
	if moved.Created == nil || moved.Created.RepeatUntil == nil ||
		!moved.Created.RepeatUntil.Equal(at("2026-11-03T00:00:00Z")) {
		t.Fatalf("new end = %v, want the day the series now starts", moved.Created)
	}
}

func TestMoveFollowingOnTheFirstOccurrenceMovesTheSeries(t *testing.T) {
	database := dbtest.NewDB(t)
	series := weeklySeries(t, database.Queries, 0)

	moved := moveSeries(t, database, series.ID, calendarutil.MoveFollowing,
		"2026-09-01T15:00:00Z", "2026-09-02T16:00:00Z", "2026-09-02T17:00:00Z")
	if moved.Created != nil {
		t.Fatal("the first occurrence started a second series")
	}
	if !moved.Series.Start.Equal(at("2026-09-02T16:00:00Z")) || !moved.Series.End.Equal(at("2026-09-02T17:00:00Z")) {
		t.Errorf("series times = %s..%s", moved.Series.Start, moved.Series.End)
	}
	if !sameTimes(moved.Series.ExcludedStarts, []string{"2026-09-16T16:00:00Z", "2026-10-21T16:00:00Z"}) {
		t.Errorf("exclusions = %v, want both shifted with the series", moved.Series.ExcludedStarts)
	}
}

func TestMoveAllShiftsExclusions(t *testing.T) {
	database := dbtest.NewDB(t)
	series := weeklySeries(t, database.Queries, 0)

	moved := moveSeries(t, database, series.ID, calendarutil.MoveAllEvents,
		"2026-10-13T15:00:00Z", "2026-10-14T16:00:00Z", "2026-10-14T17:00:00Z")
	if moved.Created != nil {
		t.Fatal("moving the series created another event")
	}
	if !moved.Series.Start.Equal(at("2026-09-02T16:00:00Z")) || !moved.Series.End.Equal(at("2026-09-02T17:00:00Z")) {
		t.Errorf("series times = %s..%s, want a day and an hour on", moved.Series.Start, moved.Series.End)
	}
	if moved.Series.RepeatUntil == nil || !moved.Series.RepeatUntil.Equal(at("2026-10-31T00:00:00Z")) {
		t.Errorf("end date = %v, want it kept", moved.Series.RepeatUntil)
	}
	if !sameTimes(moved.Series.ExcludedStarts, []string{"2026-09-16T16:00:00Z", "2026-10-21T16:00:00Z"}) {
		t.Errorf("exclusions = %v", moved.Series.ExcludedStarts)
	}
}

func TestMoveRejectsOneOccurrenceOfAOneOff(t *testing.T) {
	database := dbtest.NewDB(t)
	event := create(t, database.Queries, timed("Call", "2026-09-29T12:00:00Z", "2026-09-29T12:30:00Z"))
	_, err := calendarutil.MoveEvent(context.Background(), calendarutil.MoveEventParams{
		Database: database, ID: event.ID, Scope: calendarutil.MoveThisEvent,
		OccurrenceStart: event.Start, Start: event.Start, End: event.End.Add(time.Hour),
	})
	if !errors.Is(err, calendarutil.ErrInvalidEvent) {
		t.Fatalf("MoveEvent = %v, want an invalid event", err)
	}
}

func makeUser(t *testing.T, q db.Querier) int64 {
	t.Helper()
	user, err := q.CreateUser(context.Background(), db.CreateUserParams{
		Username: "maya", PasswordHash: "h", RecoveryPhraseHash: "r",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	return user.ID
}

func sameTimes(got []time.Time, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if !got[i].Equal(at(want[i])) {
			return false
		}
	}
	return true
}
