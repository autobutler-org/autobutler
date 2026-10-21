package authutil_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/authutil"
	"github.com/autobutler-org/quark/pkg/vfs"
)

// setupRaceRounds is how many fresh installs each engine races setup on.
const setupRaceRounds = 20

// setupRacers is how many first-boot setups are fired at once per round.
const setupRacers = 4

// TestSetup_OnlyOneFoundingAdminUnderConcurrentRequests fires several setups
// with different usernames at a fresh install at the same moment, on both
// engines. Exactly one may succeed, every other one is told setup is already
// complete, and the database holds one user, who is admin (#3647).
func TestSetup_OnlyOneFoundingAdminUnderConcurrentRequests(t *testing.T) {
	race := func(t *testing.T, database *db.DatabaseSqlc) {
		ctx := context.Background()
		errs := make([]error, setupRacers)
		start := make(chan struct{})
		var done sync.WaitGroup
		for i := range setupRacers {
			done.Go(func() {
				<-start
				_, errs[i] = authutil.Setup(ctx, authutil.SetupParams{
					Database:   database,
					Files:      vfs.NewMemVFS("files"),
					Username:   fmt.Sprintf("founder%d", i),
					AuthKey:    dbtest.AuthKey("long-enough"),
					SaltSecret: dbtest.SaltSecret,
				})
			})
		}
		close(start)
		done.Wait()

		succeeded := 0
		for _, err := range errs {
			switch {
			case err == nil:
				succeeded++
			case err.Error() != "setup already complete":
				t.Fatalf("a setup failed with %v, want nil or setup already complete; all: %v", err, errs)
			}
		}
		users, err := database.Queries.CountUsers(ctx)
		if err != nil {
			t.Fatal(err)
		}
		admins, err := database.Queries.CountActiveAdmins(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if succeeded != 1 || users != 1 || admins != 1 {
			t.Fatalf("%d setups succeeded, %d users and %d admins; want 1 of each (errors: %v)", succeeded, users, admins, errs)
		}
	}

	t.Run("sqlite", func(t *testing.T) {
		for round := range setupRaceRounds {
			t.Run(fmt.Sprint(round), func(t *testing.T) { race(t, dbtest.NewDB(t)) })
		}
	})
	t.Run("postgres", func(t *testing.T) {
		for round := range setupRaceRounds {
			t.Run(fmt.Sprint(round), func(t *testing.T) {
				database, err := db.ConnectToPostgres(dbtest.PostgresURL(t))
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { database.Db.Close() })
				race(t, database)
			})
		}
	})
}
