package db_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
)

// ConnectToDatabase is what the server calls, and the environment picks what
// it opens. The SQLite half is every other test: it is the same code it was.
func TestConnectToDatabaseOpensWhatTheEnvironmentNames(t *testing.T) {
	t.Run("an unknown backend is refused", func(t *testing.T) {
		t.Setenv(db.BackendEnv, "mysql")
		if database, err := db.ConnectToDatabase(); err == nil || !strings.Contains(err.Error(), `"sqlite"`) {
			t.Fatalf("ConnectToDatabase() = %v, %v, want a refusal naming the backends", database, err)
		}
	})
	t.Run("postgres without a URL is refused", func(t *testing.T) {
		t.Setenv(db.BackendEnv, "postgres")
		t.Setenv(db.DatabaseURLEnv, "")
		if database, err := db.ConnectToDatabase(); err == nil || !strings.Contains(err.Error(), db.DatabaseURLEnv) {
			t.Fatalf("ConnectToDatabase() = %v, %v, want a refusal naming %s", database, err, db.DatabaseURLEnv)
		}
	})
	t.Run("postgres", func(t *testing.T) {
		databaseURL := dbtest.PostgresURL(t)
		t.Setenv(db.BackendEnv, "postgres")
		t.Setenv(db.DatabaseURLEnv, databaseURL)
		database, err := db.ConnectToDatabase()
		if err != nil {
			t.Fatal(err)
		}
		defer database.Db.Close()
		if got := database.Backend(); got != db.Postgres {
			t.Errorf("Backend() = %q, want %q", got, db.Postgres)
		}
	})
}

func TestRequireSQLite(t *testing.T) {
	if err := db.RequireSQLite(dbtest.NewDB(t).Db, "Content search", 2958); err != nil {
		t.Errorf("RequireSQLite on SQLite = %v, want nil", err)
	}

	database := connectToPostgres(t, dbtest.PostgresURL(t))
	err := db.RequireSQLite(database.Db, "Content search", 2958)
	var unsupported *db.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("RequireSQLite on PostgreSQL = %v, want a *db.UnsupportedError", err)
	}
	if want := "Content search is not supported on the PostgreSQL backend yet (#2958)"; err.Error() != want {
		t.Errorf("the refusal reads %q, want %q", err, want)
	}
	if got := unsupported.HTTPStatus(); got != http.StatusNotImplemented {
		t.Errorf("HTTPStatus() = %d, want %d", got, http.StatusNotImplemented)
	}
}

// The reset an admin asks for works on PostgreSQL as it does on SQLite:
// everything goes but the settings and the account request history, a new
// install id is issued, and the pool that was open before it keeps working.
func TestResetDatabaseOnPostgres(t *testing.T) {
	databaseURL := dbtest.PostgresURL(t)
	database := connectToPostgres(t, databaseURL)
	ctx := context.Background()
	q := database.Queries

	installID, err := q.GetInstallID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.CreateUser(ctx, db.CreateUserParams{Username: "ada"}); err != nil {
		t.Fatal(err)
	}
	if err := q.SetSetting(ctx, db.SetSettingParams{Key: "deviceId", Value: `"abc"`}); err != nil {
		t.Fatal(err)
	}
	decided := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	for _, username := range []string{"first", "second"} {
		if err := q.AddAccountRequestDecision(ctx, db.AddAccountRequestDecisionParams{
			Username: username, Outcome: "approved", DecidedBy: "ada", DecidedAt: decided,
		}); err != nil {
			t.Fatal(err)
		}
	}

	if err := db.ResetDatabase(database); err != nil {
		t.Fatalf("ResetDatabase: %v", err)
	}

	if _, err := q.GetUserByUsername(ctx, "ada"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("after the reset the account is still there: %v", err)
	}
	if got, err := q.GetInstallID(ctx); err != nil || got == "" || got == installID {
		t.Errorf("after the reset the install id is %q (%v), want a new one in place of %q", got, err, installID)
	}
	if got, err := q.GetSetting(ctx, "deviceId"); err != nil || got != `"abc"` {
		t.Errorf("after the reset the setting is %q (%v), want it kept", got, err)
	}
	// The history is kept, and its id sequence was moved past the kept rows:
	// AddAccountRequestDecision is ON CONFLICT DO NOTHING, so a third decision
	// handed an id a kept row holds would vanish without an error.
	if err := q.AddAccountRequestDecision(ctx, db.AddAccountRequestDecisionParams{
		Username: "third", Outcome: "denied", DecidedBy: "ada", DecidedAt: decided,
	}); err != nil {
		t.Fatal(err)
	}
	history, err := q.ListAccountRequestHistory(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	usernames := []string{}
	for _, row := range history {
		usernames = append(usernames, row.Username)
	}
	if got := strings.Join(usernames, ","); got != "third,second,first" {
		t.Errorf("after the reset the request history is %q, want third,second,first", got)
	}
	// The seed rows came back with the schema.
	if groups, err := q.ListGroups(ctx); err != nil || len(groups) != 1 {
		t.Errorf("after the reset there are %d groups (%v), want the built-in one", len(groups), err)
	}

	// The schema version is untouched, so the next start has nothing to do
	// and finds the database it left.
	var version int
	var dirty bool
	if err := database.Db.QueryRow(`SELECT version, dirty FROM schema_migrations`).Scan(&version, &dirty); err != nil || dirty {
		t.Fatalf("schema_migrations = %d, dirty %v, %v", version, dirty, err)
	}
	again := connectToPostgres(t, databaseURL)
	if got, err := again.Queries.GetSetting(ctx, "deviceId"); err != nil || got != `"abc"` {
		t.Errorf("after a restart the setting is %q (%v)", got, err)
	}
}
