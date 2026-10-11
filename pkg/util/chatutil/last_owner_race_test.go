package chatutil_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/autobutler-org/quark/internal/db"
	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/chatutil"
)

// lastOwnerRaceRounds is how many times the pair of requests is fired.
const lastOwnerRaceRounds = 100

// TestLastOwnerGuard_HoldsUnderConcurrentRequests has a channel's two holders
// of manage_channel leave at the same moment, with a third member staying, on
// both engines. Exactly one may leave, the other is refused with ErrLastOwner,
// and the channel keeps a holder (#3171).
func TestLastOwnerGuard_HoldsUnderConcurrentRequests(t *testing.T) {
	race := func(t *testing.T, database *db.DatabaseSqlc) {
		f := newFixtureOn(t, database)
		channel := f.create(t, "bob", "design")
		if err := f.set(t, "bob", channel.ID, f.users["dave"], 0, chatutil.PresetMember); err != nil {
			t.Fatal(err)
		}
		leavers := []string{"bob", "carol"}
		for round := range lastOwnerRaceRounds {
			for _, name := range leavers {
				if err := f.set(t, "admin", channel.ID, f.users[name], 0, chatutil.PresetOwner); err != nil {
					t.Fatal(err)
				}
			}
			errs := make([]error, 2)
			start := make(chan struct{})
			var done sync.WaitGroup
			for i, name := range leavers {
				done.Go(func() {
					<-start
					errs[i] = f.remove(name, channel.ID, f.users[name], 0)
				})
			}
			close(start)
			done.Wait()

			holders := 0
			for _, name := range leavers {
				if f.resolve(t, name, channel.ID).Has(chatutil.PermManageChannel) {
					holders++
				}
			}
			refused := errs[0]
			if refused == nil {
				refused = errs[1]
			}
			if (errs[0] == nil) == (errs[1] == nil) || !errors.Is(refused, chatutil.ErrLastOwner) || holders != 1 {
				t.Fatalf("round %d: bob = %v, carol = %v, %d holders of manage_channel left; want one success, one ErrLastOwner and 1 holder",
					round, errs[0], errs[1], holders)
			}
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
