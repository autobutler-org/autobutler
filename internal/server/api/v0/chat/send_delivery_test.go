package v0_chat_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/chatutil"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
)

// The send-delivery guarantees a chat client leans on (#3477): a retry does
// not store a second copy, everyone reads one server order, and a message id
// reaches a reader once however often its event and its catch-up overlap.

// heardIDs lists the message ids of the chat_message_created events heard.
func heardIDs(heard []eventbus.ChatMessageChanged) []int64 {
	out := make([]int64, 0, len(heard))
	for _, h := range heard {
		out = append(out, h.MessageID)
	}
	return out
}

// pageAll walks a channel forward from the beginning in pages of limit, the
// way a client catches up from a cursor, and returns every id in order.
func (h harness) pageAll(t *testing.T, messages, as string, limit int) []int64 {
	t.Helper()
	var all []int64
	for cursor := int64(0); ; {
		var page chatutil.ListMessagesResult
		h.expect(t, http.StatusOK, http.MethodGet, messages+"?after="+strconv.FormatInt(cursor, 10)+"&limit="+strconv.Itoa(limit), as, "", &page)
		if len(page.Messages) == 0 {
			return all
		}
		all = append(all, ids(page)...)
		cursor = all[len(all)-1]
	}
}

// TestChatDelivery_LostResponseRetry stores a post whose response the client
// never sees and retries it byte for byte: carol, reading along, sees one
// message and hears one chat_message_created.
func TestChatDelivery_LostResponseRetry(t *testing.T) {
	h := newHarness(t)
	_, room := roomWithKey(t, h)
	messages := room + "/messages"
	h.expect(t, http.StatusOK, http.MethodPut, room+"/members", "bob", userBody(h.users["carol"], viewer...), nil)
	h.drainMessages()

	// The first response is lost: bob's client never reads it.
	if code, _ := h.do(t, http.MethodPost, messages, "bob", messageBody('h', 64, 1)); code != http.StatusCreated {
		t.Fatalf("first post = %d", code)
	}
	var retried chatutil.Message
	h.expect(t, http.StatusOK, http.MethodPost, messages, "bob", messageBody('h', 64, 1), &retried)

	var page chatutil.ListMessagesResult
	h.expect(t, http.StatusOK, http.MethodGet, messages+"?after=0", "carol", "", &page)
	if got := ids(page); len(got) != 1 || got[0] != retried.ID {
		t.Errorf("carol reads %v, want only the retried message %d", got, retried.ID)
	}
	if heard := heardIDs(h.drainMessages()); len(heard) != 1 || heard[0] != retried.ID {
		t.Errorf("heard %v, want one event for %d", heard, retried.ID)
	}
}

// withClientID adds clientMessageId to a messageBody.
func withClientID(body, id string) string {
	return strings.TrimSuffix(body, "}") + `,"clientMessageId":"` + id + `"}`
}

// TestChatDelivery_RetryEncryptedAgainStoresOnce is the retry the app actually
// makes: ChatController.retry encrypts the text again, so the second post has
// a fresh nonce and the nonce replay check of #2487 cannot see it is the same
// message. The client message id both posts carry does (#3531): the retry is
// answered with the first post and announced to nobody.
func TestChatDelivery_RetryEncryptedAgainStoresOnce(t *testing.T) {
	h := newHarness(t)
	_, room := roomWithKey(t, h)
	messages := room + "/messages"
	h.drainMessages()

	// The same "hi" under two nonces: the first response lost, then the
	// retry.
	var first, retried chatutil.Message
	h.expect(t, http.StatusCreated, http.MethodPost, messages, "bob", withClientID(messageBody('h', 64, 1), "send-1"), &first)
	h.expect(t, http.StatusOK, http.MethodPost, messages, "bob", withClientID(messageBody('i', 64, 1), "send-1"), &retried)
	if retried.ID != first.ID || string(retried.Ciphertext) != string(first.Ciphertext) {
		t.Errorf("retry answered %d %q, want the first post %d %q", retried.ID, retried.Ciphertext, first.ID, first.Ciphertext)
	}

	var page chatutil.ListMessagesResult
	h.expect(t, http.StatusOK, http.MethodGet, messages+"?after=0", "bob", "", &page)
	if len(page.Messages) != 1 {
		t.Errorf("channel holds %d messages after one retried send, want 1", len(page.Messages))
	}
	if heard := heardIDs(h.drainMessages()); !slices.Equal(heard, []int64{first.ID}) {
		t.Errorf("heard %v, want one event for %d", heard, first.ID)
	}
}

// TestChatDelivery_ClientIDScope: a client message id names one author's send
// in one channel. Another id is another message, another author may use the
// same id, a retry of a deleted send stays deleted, and an id longer than any
// client makes is refused.
func TestChatDelivery_ClientIDScope(t *testing.T) {
	h := newHarness(t)
	_, room := roomWithKey(t, h)
	messages := room + "/messages"
	h.expect(t, http.StatusOK, http.MethodPut, room+"/members", "bob", userBody(h.users["carol"], member...), nil)

	var a, b, carols chatutil.Message
	h.expect(t, http.StatusCreated, http.MethodPost, messages, "bob", withClientID(messageBody('a', 64, 1), "one"), &a)
	h.expect(t, http.StatusCreated, http.MethodPost, messages, "bob", withClientID(messageBody('b', 64, 1), "two"), &b)
	h.expect(t, http.StatusCreated, http.MethodPost, messages, "carol", withClientID(messageBody('c', 64, 1), "one"), &carols)
	if a.ID == b.ID || a.ID == carols.ID {
		t.Fatalf("ids %d, %d, %d; want three messages", a.ID, b.ID, carols.ID)
	}

	h.expect(t, http.StatusOK, http.MethodDelete, "/chat/messages/"+strconv.FormatInt(b.ID, 10), "bob", "", nil)
	var again chatutil.Message
	h.expect(t, http.StatusOK, http.MethodPost, messages, "bob", withClientID(messageBody('d', 64, 1), "two"), &again)
	if again.ID != b.ID || again.DeletedAt == nil {
		t.Errorf("retry of a deleted send = %+v, want tombstone %d", again, b.ID)
	}

	h.expect(t, http.StatusBadRequest, http.MethodPost, messages, "bob", withClientID(messageBody('e', 64, 1), strings.Repeat("x", 65)), nil)
	var page chatutil.ListMessagesResult
	h.expect(t, http.StatusOK, http.MethodGet, messages+"?after=0", "bob", "", &page)
	if got := ids(page); !slices.Equal(got, []int64{a.ID, b.ID, carols.ID}) {
		t.Errorf("channel holds %v, want %v", got, []int64{a.ID, b.ID, carols.ID})
	}
}

// TestChatDelivery_InterleavedOrder has bob and carol each send 20 messages at
// once. Both read the same ascending order, whole or paged either way, each
// sender's messages keep the order they were sent in, and every id is
// announced exactly once.
func TestChatDelivery_InterleavedOrder(t *testing.T) {
	h := newHarness(t)
	_, room := roomWithKey(t, h)
	messages := room + "/messages"
	h.expect(t, http.StatusOK, http.MethodPut, room+"/members", "bob", userBody(h.users["carol"], member...), nil)
	h.drainMessages()

	const each = 20
	sent := map[string][]int64{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	start := make(chan struct{})
	for as, base := range map[string]byte{"bob": 'a', "carol": 'A'} {
		wg.Go(func() {
			<-start
			for i := range each {
				code, body := h.do(t, http.MethodPost, messages, as, messageBody(base+byte(i), 64, 1))
				if code != http.StatusCreated {
					t.Errorf("%s's post %d = %d %s", as, i, code, body)
					return
				}
				var m chatutil.Message
				if err := json.Unmarshal([]byte(body), &m); err != nil {
					t.Errorf("%s's post %d: %v", as, i, err)
					return
				}
				mu.Lock()
				sent[as] = append(sent[as], m.ID)
				mu.Unlock()
			}
		})
	}
	close(start)
	wg.Wait()
	if t.Failed() {
		t.FailNow()
	}

	var bobs, carols chatutil.ListMessagesResult
	h.expect(t, http.StatusOK, http.MethodGet, messages+"?after=0&limit=200", "bob", "", &bobs)
	h.expect(t, http.StatusOK, http.MethodGet, messages+"?after=0&limit=200", "carol", "", &carols)
	order := ids(bobs)
	if !slices.Equal(order, ids(carols)) {
		t.Fatalf("bob reads %v, carol reads %v", order, ids(carols))
	}
	if len(order) != 2*each || !slices.IsSorted(order) || len(slices.Compact(slices.Clone(order))) != len(order) {
		t.Fatalf("order = %v, want %d distinct ascending ids", order, 2*each)
	}
	for as, mine := range sent {
		if !slices.IsSorted(mine) {
			t.Errorf("%s's messages were stored out of send order: %v", as, mine)
		}
	}

	// Paging forward from a cursor, and backward from the newest, read the
	// same order with no gap or overlap.
	if paged := h.pageAll(t, messages, "carol", 7); !slices.Equal(paged, order) {
		t.Errorf("forward pages = %v, want %v", paged, order)
	}
	var backward []int64
	for before := int64(0); ; {
		path := messages + "?limit=7"
		if before != 0 {
			path += "&before=" + strconv.FormatInt(before, 10)
		}
		var page chatutil.ListMessagesResult
		h.expect(t, http.StatusOK, http.MethodGet, path, "bob", "", &page)
		if len(page.Messages) == 0 {
			break
		}
		backward = append(ids(page), backward...)
		before = backward[0]
	}
	if !slices.Equal(backward, order) {
		t.Errorf("backward pages = %v, want %v", backward, order)
	}

	// Live events may interleave the two senders differently from the
	// stored order, so a client sorts by id; what the bus owes it is each id
	// once, and each sender's in the order sent.
	heard := h.drainMessages()
	if got := slices.Sorted(slices.Values(heardIDs(heard))); !slices.Equal(got, order) {
		t.Errorf("heard %v, want each of %v once", got, order)
	}
	byAuthor := map[int64][]int64{}
	for _, evt := range heard {
		author := evt.Message.(chatutil.Message).AuthorID
		byAuthor[author] = append(byAuthor[author], evt.MessageID)
	}
	for as, mine := range sent {
		if got := byAuthor[h.users[as]]; !slices.Equal(got, mine) {
			t.Errorf("%s's events arrived as %v, sent as %v", as, got, mine)
		}
	}
}

// TestChatDelivery_ResyncOverlap extends TestChatMessages_BurstCatchUp from
// "nothing is missing" to "nothing is doubled": a socket hears a few
// messages, falls behind a burst and gets a resync with the tail of the
// burst after it, and catches up from the last id it rendered. The catch-up
// is exactly what it missed, the events after the resync are a subset of it
// to merge by id, a byte-identical retry announces nothing, and a catch-up
// from too far back still returns each id once.
func TestChatDelivery_ResyncOverlap(t *testing.T) {
	h := newHarness(t)
	_, room := roomWithKey(t, h)
	messages := room + "/messages"
	h.drainMessages()
	socket, unsub := h.deps.EventBus().SubscribeLossy("resync-overlap")
	defer unsub()

	post := func(fill byte) int64 {
		t.Helper()
		var m chatutil.Message
		h.expect(t, http.StatusCreated, http.MethodPost, messages, "bob", messageBody(fill, 64, 1), &m)
		return m.ID
	}
	// listen reads what the socket holds: the message ids before any resync,
	// whether one came, and the ids after it.
	listen := func() (before []int64, resynced bool, after []int64) {
		for {
			select {
			case evt := <-socket:
				switch data, ok := evt.Data.(eventbus.ChatMessageChanged); {
				case evt.Kind == eventbus.EventResync:
					resynced = true
				case ok && evt.Kind == eventbus.EventChatMessageCreated && resynced:
					after = append(after, data.MessageID)
				case ok && evt.Kind == eventbus.EventChatMessageCreated:
					before = append(before, data.MessageID)
				}
			default:
				return before, resynced, after
			}
		}
	}

	var posted []int64
	for i := range 3 {
		posted = append(posted, post(byte('a'+i)))
	}
	rendered, resynced, _ := listen()
	if resynced || !slices.Equal(rendered, posted) {
		t.Fatalf("live = %v (resync %v), want %v", rendered, resynced, posted)
	}
	cursor := rendered[len(rendered)-1]
	// A lost-response retry of the last one is answered, not announced.
	h.expect(t, http.StatusOK, http.MethodPost, messages, "bob", messageBody('c', 64, 1), nil)

	const burst = 20
	for i := range burst {
		posted = append(posted, post(byte('A'+i)))
	}
	early, resynced, tail := listen()
	if !resynced || len(early) != 0 || len(tail) == 0 {
		t.Fatalf("after the burst: %v, resync %v, then %v; want only a resync and the burst's tail", early, resynced, tail)
	}
	var page chatutil.ListMessagesResult
	h.expect(t, http.StatusOK, http.MethodGet, messages+"?after="+strconv.FormatInt(cursor, 10), "bob", "", &page)
	missed := ids(page)
	if !slices.Equal(missed, posted[3:]) {
		t.Fatalf("catch-up after %d = %v, want %v", cursor, missed, posted[3:])
	}
	for _, id := range tail {
		if !slices.Contains(missed, id) {
			t.Errorf("event %d after the resync is not in the catch-up page", id)
		}
	}
	timeline := slices.Concat(rendered, missed)
	for _, id := range tail {
		if !slices.Contains(timeline, id) {
			timeline = append(timeline, id)
		}
	}
	if !slices.Equal(timeline, posted) {
		t.Errorf("merged timeline = %v, want %v", timeline, posted)
	}
	if again := h.pageAll(t, messages, "bob", 200); !slices.Equal(again, posted) {
		t.Errorf("catch-up from the beginning = %v, want %v", again, posted)
	}
	if heard := heardIDs(h.drainMessages()); !slices.Equal(heard, posted) {
		t.Errorf("published %v, want each of %v once, the retry not at all", heard, posted)
	}
}
