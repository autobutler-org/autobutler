package db_test

import (
	"context"
	"database/sql"
	"reflect"
	"sort"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
)

// jobNames lists the names of the jobs returned, sorted.
func jobNames(jobs []db.Job) []string {
	names := []string{}
	for _, job := range jobs {
		names = append(names, job.Name)
	}
	sort.Strings(names)
	return names
}

// ListJobs and ListUserJobs filter by an IN-list sqlc expands at call time.
// Preparing them proves nothing about what the expansion binds: a parameter
// beside the slice can end up bound to one of the kinds (sqlc-dev/sqlc#2452),
// and an empty slice takes a different path (sqlc-dev/sqlc#3327). So this
// runs them, with a kind that is a prefix of another to catch a filter that
// matches by anything looser than equality.
func TestListJobsFiltersByKindAndUser(t *testing.T) {
	ctx := context.Background()
	q := dbtest.NewDB(t).Queries
	users := map[string]sql.NullInt64{}
	for _, name := range []string{"ann", "bob"} {
		user, err := q.CreateUser(ctx, db.CreateUserParams{Username: name, PasswordHash: "h", RecoveryPhraseHash: "r"})
		if err != nil {
			t.Fatal(err)
		}
		users[name] = sql.NullInt64{Int64: user.ID, Valid: true}
		for _, kind := range []string{"scan", "scan.deep", "backup"} {
			if _, err := q.CreateJob(ctx, db.CreateJobParams{Kind: kind, Name: name + ":" + kind, UserID: users[name]}); err != nil {
				t.Fatal(err)
			}
		}
	}

	for _, tc := range []struct {
		user  string
		kinds []string
		want  []string
	}{
		{"ann", []string{"scan", "backup"}, []string{"ann:backup", "ann:scan"}},
		{"bob", []string{"scan", "backup"}, []string{"bob:backup", "bob:scan"}},
		{"ann", []string{"scan"}, []string{"ann:scan"}},
		{"ann", []string{"scan.deep", "scan", "backup"}, []string{"ann:backup", "ann:scan", "ann:scan.deep"}},
		{"ann", []string{"sca", "nope"}, []string{}},
		{"ann", []string{}, []string{}},
		{"ann", nil, []string{}},
	} {
		jobs, err := q.ListUserJobs(ctx, db.ListUserJobsParams{UserID: users[tc.user], Kinds: tc.kinds})
		if err != nil {
			t.Fatalf("ListUserJobs(%s, %q): %v", tc.user, tc.kinds, err)
		}
		if got := jobNames(jobs); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ListUserJobs(%s, %q) = %v, want %v", tc.user, tc.kinds, got, tc.want)
		}
	}

	for _, tc := range []struct {
		kinds []string
		want  []string
	}{
		{[]string{"scan", "backup"}, []string{"ann:backup", "ann:scan", "bob:backup", "bob:scan"}},
		{[]string{"scan"}, []string{"ann:scan", "bob:scan"}},
		{[]string{}, []string{}},
	} {
		jobs, err := q.ListJobs(ctx, tc.kinds)
		if err != nil {
			t.Fatalf("ListJobs(%q): %v", tc.kinds, err)
		}
		if got := jobNames(jobs); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ListJobs(%q) = %v, want %v", tc.kinds, got, tc.want)
		}
	}
}
