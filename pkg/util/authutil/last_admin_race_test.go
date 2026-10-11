package authutil_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/authutil"
)

// lastAdminRaceRounds is how many times each pair of requests is fired.
// Before the guard took a lock, PostgreSQL lost both admins in nearly every
// round (#3171).
const lastAdminRaceRounds = 100

// TestLastAdminGuard_HoldsUnderConcurrentRequests fires the two requests that
// would each take away one of the two active admins at the same moment, on
// both engines. Exactly one may succeed, the other is refused with
// ErrLastAdmin, and an active admin remains (#3171).
func TestLastAdminGuard_HoldsUnderConcurrentRequests(t *testing.T) {
	changes := map[string]func(database *db.DatabaseSqlc, username string) error{
		"disable": func(database *db.DatabaseSqlc, username string) error {
			_, err := authutil.DisableUser(context.Background(), authutil.DisableUserParams{Database: database, Username: username})
			return err
		},
		"demote": func(database *db.DatabaseSqlc, username string) error {
			return authutil.DemoteFromAdmin(context.Background(), database, username)
		},
	}
	race := func(t *testing.T, database *db.DatabaseSqlc) {
		mkUser(t, database.Queries, "a", true)
		mkUser(t, database.Queries, "b", true)
		for name, change := range changes {
			t.Run(name, func(t *testing.T) {
				for round := range lastAdminRaceRounds {
					if _, err := database.Db.Exec(`UPDATE users SET is_admin = 1, status = 'active'`); err != nil {
						t.Fatal(err)
					}
					errs := make([]error, 2)
					start := make(chan struct{})
					var done sync.WaitGroup
					for i, username := range []string{"a", "b"} {
						done.Go(func() {
							<-start
							errs[i] = change(database, username)
						})
					}
					close(start)
					done.Wait()

					admins, err := database.Queries.CountActiveAdmins(context.Background())
					if err != nil {
						t.Fatal(err)
					}
					refused := errs[0]
					if refused == nil {
						refused = errs[1]
					}
					if (errs[0] == nil) == (errs[1] == nil) || !errors.Is(refused, authutil.ErrLastAdmin) || admins != 1 {
						t.Fatalf("round %d: a = %v, b = %v, %d active admins left; want one success, one ErrLastAdmin and 1 admin",
							round, errs[0], errs[1], admins)
					}
				}
			})
		}
	}

	t.Run("sqlite", func(t *testing.T) { race(t, dbtest.NewDB(t)) })
	t.Run("postgres", func(t *testing.T) {
		database, err := db.ConnectToPostgres(dbtest.PostgresURL(t))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { database.Db.Close() })
		race(t, database)
	})
}
