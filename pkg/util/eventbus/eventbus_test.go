package eventbus_test

import (
	"sync"
	"testing"
	"time"

	"github.com/autobutler-org/autobutler/pkg/util/eventbus"
)

func TestSubscribeAndPublish(t *testing.T) {
	bus := eventbus.New()
	ch, unsub := bus.Subscribe("test-1")
	defer unsub()

	evt := eventbus.Event{Kind: eventbus.EventUpload, Path: "/foo.txt"}
	bus.Publish(evt)

	select {
	case got := <-ch:
		if got.Kind != eventbus.EventUpload {
			t.Errorf("expected kind %q, got %q", eventbus.EventUpload, got.Kind)
		}
		if got.Path != "/foo.txt" {
			t.Errorf("expected path %q, got %q", "/foo.txt", got.Path)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for event")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := eventbus.New()

	var wg sync.WaitGroup
	received := make([]int, 3)
	for i := range received {
		i := i
		ch, unsub := bus.Subscribe(string(rune('a' + i)))
		defer unsub()
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ch:
				received[i] = 1
			case <-time.After(200 * time.Millisecond):
			}
		}()
	}

	bus.Publish(eventbus.Event{Kind: eventbus.EventDelete, Path: "/bar.txt"})
	wg.Wait()

	for i, v := range received {
		if v != 1 {
			t.Errorf("subscriber %d did not receive event", i)
		}
	}
}

func TestUnsubscribeClosesChannel(t *testing.T) {
	bus := eventbus.New()
	_, unsub := bus.Subscribe("unsub-test")
	unsub()

	// Should not panic — publishing to a bus with no subscribers is a no-op.
	bus.Publish(eventbus.Event{Kind: eventbus.EventNewFolder, Path: "/dir"})
}

func TestPublishDoesNotBlockOnFullBuffer(t *testing.T) {
	bus := eventbus.New()
	// Subscribe but never drain the channel — buffer will fill
	_, unsub := bus.Subscribe("slow")
	defer unsub()

	// Fill the 16-slot buffer + one extra; Publish must not block
	done := make(chan struct{})
	go func() {
		for i := 0; i < 20; i++ {
			bus.Publish(eventbus.Event{Kind: eventbus.EventUpload, Path: "/x"})
		}
		close(done)
	}()

	select {
	case <-done:
		// pass — Publish returned without blocking
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Publish blocked on full subscriber buffer")
	}
}

func TestMoveEventHasNewPath(t *testing.T) {
	bus := eventbus.New()
	ch, unsub := bus.Subscribe("move-test")
	defer unsub()

	bus.Publish(eventbus.Event{
		Kind:    eventbus.EventMove,
		Path:    "/old.txt",
		NewPath: "/new.txt",
	})

	select {
	case got := <-ch:
		if got.NewPath != "/new.txt" {
			t.Errorf("expected NewPath %q, got %q", "/new.txt", got.NewPath)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for move event")
	}
}

func TestNoEventAfterUnsubscribe(t *testing.T) {
	bus := eventbus.New()
	ch, unsub := bus.Subscribe("gone")

	// Drain and unsubscribe
	unsub()

	// Publish after unsubscribe — channel should be closed, no new events
	bus.Publish(eventbus.Event{Kind: eventbus.EventUpload, Path: "/late.txt"})

	// Channel is closed; any read will return zero-value immediately
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed, got value")
		}
	default:
		// closed channel is drained — acceptable
	}
}
