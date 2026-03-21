package eventbus

import "sync"

type EventKind string

const (
	EventUpload    EventKind = "upload"
	EventDelete    EventKind = "delete"
	EventMove      EventKind = "move"
	EventNewFolder EventKind = "new_folder"
)

type Event struct {
	Kind    EventKind `json:"kind"`
	Path    string    `json:"path"`              // affected path
	NewPath string    `json:"newPath,omitempty"` // for move
}

type Bus struct {
	mu          sync.RWMutex
	subscribers map[string]chan Event
}

func New() *Bus { return &Bus{subscribers: map[string]chan Event{}} }

// Subscribe registers a subscriber and returns its channel and an unsubscribe func.
func (b *Bus) Subscribe(id string) (<-chan Event, func()) {
	ch := make(chan Event, 16)
	b.mu.Lock()
	b.subscribers[id] = ch
	b.mu.Unlock()
	return ch, func() {
		b.mu.Lock()
		delete(b.subscribers, id)
		close(ch)
		b.mu.Unlock()
	}
}

// Publish sends an event to all subscribers (non-blocking; drops if buffer full).
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers {
		select {
		case ch <- e:
		default:
		}
	}
}
