package chatutil_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/autobutler-org/quark/internal/db/dbtest"
	"github.com/autobutler-org/quark/pkg/util/chatutil"
)

// TestCreateKeyVersion_ConcurrentLoserIsVersionConflict has two members create
// version 1 of a keyless channel at the same moment: one wins and the other
// gets ErrVersionConflict, never a SQLITE_BUSY (#3188).
func TestCreateKeyVersion_ConcurrentLoserIsVersionConflict(t *testing.T) {
	for round := range 100 {
		f := newFixtureOn(t, dbtest.NewDB(t))
		f.publishKeys(t, "bob", "carol")
		channel := f.create(t, "bob", "design")
		if err := f.set(t, "bob", channel.ID, f.users["carol"], 0, chatutil.PresetMember); err != nil {
			t.Fatal(err)
		}
		names := []string{"bob", "carol"}
		errs := make([]error, len(names))
		start := make(chan struct{})
		var done sync.WaitGroup
		for i, name := range names {
			done.Go(func() {
				<-start
				_, errs[i] = f.createVersion(name, channel.ID, 1)
			})
		}
		close(start)
		done.Wait()

		won := 0
		for _, err := range errs {
			switch {
			case err == nil:
				won++
			case !errors.Is(err, chatutil.ErrVersionConflict):
				t.Fatalf("round %d: %v; want nil or ErrVersionConflict", round, err)
			}
		}
		if won != 1 {
			t.Fatalf("round %d: %d winners (%v); want 1", round, won, errs)
		}
	}
}
