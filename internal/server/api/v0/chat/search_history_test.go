package v0_chat_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/chatutil"
)

// The history pages an on-device search walks when it loads older messages
// (#3479). The Quark stores ciphertext and has no search query (#2429): a
// page covers every message once, the page past the start is an empty list,
// and a room the caller cannot read contributes nothing.
//
// Not covered here, because the server never sees the text: the search bar's
// match list and no-results line, case and unicode folding, a pending send,
// a locked chat dropping the query, and clearing the query when the channel,
// the lock, or a resync changes. TestChatSearch_PlaintextQuery names that gap.

// postMessages stores n messages in order. Each has its own fill byte, so
// each ciphertext is a different message.
func postMessages(t *testing.T, h harness, messages string, n int) []chatutil.Message {
	t.Helper()
	posted := make([]chatutil.Message, 0, n)
	for i := range n {
		var m chatutil.Message
		h.expect(t, http.StatusCreated, http.MethodPost, messages, "bob", messageBody('a'+byte(i), 64, 1), &m)
		posted = append(posted, m)
	}
	return posted
}

// olderPages walks messages the way ChatMessagesController.loadOlder does:
// the newest page of limit, then the page before the oldest id on it, until
// a page comes back empty. A repeated id, a page that is not oldest-first,
// or a walk that does not end fails the test.
func (h harness) olderPages(t *testing.T, messages, as string, limit int) []chatutil.ListMessagesResult {
	t.Helper()
	var pages []chatutil.ListMessagesResult
	seen := map[int64]struct{}{}
	before := int64(0)
	for {
		path := messages + "?limit=" + strconv.Itoa(limit)
		if before != 0 {
			path += "&before=" + strconv.FormatInt(before, 10)
		}
		code, body := h.do(t, http.MethodGet, path, as, "")
		if code != http.StatusOK {
			t.Fatalf("GET %s as %s = %d %s, want 200", path, as, code, body)
		}
		var page chatutil.ListMessagesResult
		if err := json.Unmarshal([]byte(body), &page); err != nil {
			t.Fatalf("decode %s: %v", body, err)
		}
		if len(page.Messages) == 0 {
			if page.Messages == nil || !strings.Contains(body, `"messages":[]`) || strings.Contains(body, `"error"`) {
				t.Fatalf("page before %d = %s, want {\"messages\":[]}", before, body)
			}
			return pages
		}
		if len(page.Messages) > limit {
			t.Fatalf("page has %d messages, limit is %d", len(page.Messages), limit)
		}
		got := ids(page)
		if !slices.IsSorted(got) {
			t.Fatalf("page %v is not oldest first", got)
		}
		oldest := got[0]
		if before != 0 && oldest >= before {
			t.Fatalf("page oldest %d is not before %d", oldest, before)
		}
		for _, id := range got {
			if _, ok := seen[id]; ok {
				t.Fatalf("message %d came back on two pages", id)
			}
			seen[id] = struct{}{}
		}
		pages = append(pages, page)
		before = oldest
		if len(pages) > 20 {
			t.Fatal("paging did not reach the start of the channel")
		}
	}
}

// emptyMessages fails unless path answers 200 with an empty message list.
func (h harness) emptyMessages(t *testing.T, path, as string) string {
	t.Helper()
	code, body := h.do(t, http.MethodGet, path, as, "")
	if code != http.StatusOK || !strings.Contains(body, `"messages":[]`) || strings.Contains(body, `"error"`) {
		t.Fatalf("GET %s as %s = %d %s, want 200 {\"messages\":[]}", path, as, code, body)
	}
	return body
}

// refuseMessages fails unless path answers want and the body does not carry
// secret, a message's base64 ciphertext.
func (h harness) refuseMessages(t *testing.T, path, as, secret string, want int) {
	t.Helper()
	code, body := h.do(t, http.MethodGet, path, as, "")
	if code != want || strings.Contains(body, secret) {
		t.Errorf("GET %s as %s = %d %s, want %d without the ciphertext", path, as, code, body, want)
	}
}

// TestChatSearch_PagesCoverEveryMessageOnce posts more than two pages, deletes
// one in the middle, and walks backward the way load-older does. Page 1 is
// the newest two, page 3 the oldest two, every id appears once, a repeated
// load-older is the same page, and the page past the start is an empty list.
func TestChatSearch_PagesCoverEveryMessageOnce(t *testing.T) {
	h := newHarness(t)
	_, room := roomWithKey(t, h)
	messages := room + "/messages"
	h.expect(t, http.StatusOK, http.MethodPut, room+"/members", "bob", userBody(h.users["carol"], viewer...), nil)

	const limit = 2
	posted := postMessages(t, h, messages, 6)
	h.expect(t, http.StatusOK, http.MethodDelete, "/chat/messages/"+strconv.FormatInt(posted[3].ID, 10), "bob", "", nil)

	// Pages are newest first. Page 1 is the two newest messages, page 3 the
	// two oldest: a hit two pages back, and the newest hit, each once. The
	// deleted message stays on page 2 as a tombstone, so the cursor does
	// not skip its neighbors.
	want := [][]int64{
		{posted[4].ID, posted[5].ID},
		{posted[2].ID, posted[3].ID},
		{posted[0].ID, posted[1].ID},
	}
	pages := h.olderPages(t, messages, "carol", limit)
	if len(pages) != len(want) {
		t.Fatalf("carol walked %d pages, want %d", len(pages), len(want))
	}
	for i, page := range pages {
		if !slices.Equal(ids(page), want[i]) {
			t.Errorf("page %d = %v, want %v", i+1, ids(page), want[i])
		}
	}
	tomb := pages[1].Messages[1]
	if tomb.ID != posted[3].ID || tomb.DeletedAt == nil || len(tomb.Ciphertext) != 0 {
		t.Errorf("deleted message on page 2 = %+v, want a tombstone", tomb)
	}

	bob := h.olderPages(t, messages, "bob", limit)
	if len(bob) != len(pages) {
		t.Fatalf("bob walked %d pages, carol walked %d", len(bob), len(pages))
	}
	for i := range pages {
		if !slices.Equal(ids(bob[i]), ids(pages[i])) {
			t.Errorf("bob's page %d = %v, carol's = %v", i+1, ids(bob[i]), ids(pages[i]))
		}
	}

	// The same cursor again is the same page, and the newest page is still
	// the newest page after the walk.
	var again, newest chatutil.ListMessagesResult
	cursor := strconv.FormatInt(pages[0].Messages[0].ID, 10)
	h.expect(t, http.StatusOK, http.MethodGet, messages+"?limit=2&before="+cursor, "carol", "", &again)
	if !slices.Equal(ids(again), want[1]) {
		t.Errorf("repeated load-older = %v, want %v", ids(again), want[1])
	}
	h.expect(t, http.StatusOK, http.MethodGet, messages+"?limit=2", "carol", "", &newest)
	if !slices.Equal(ids(newest), want[0]) {
		t.Errorf("newest page after the walk = %v, want %v", ids(newest), want[0])
	}
}

// TestChatSearch_NoResultIsAnEmptyList checks that an empty channel, and a
// page past either end of one message, answer 200 with an empty list.
func TestChatSearch_NoResultIsAnEmptyList(t *testing.T) {
	h := newHarness(t)
	_, room := roomWithKey(t, h)
	messages := room + "/messages"

	h.emptyMessages(t, messages, "bob")
	h.emptyMessages(t, messages+"?after=0", "bob")
	h.emptyMessages(t, messages+"?limit=2", "bob")

	var posted chatutil.Message
	h.expect(t, http.StatusCreated, http.MethodPost, messages, "bob", messageBody('m', 64, 1), &posted)
	id := strconv.FormatInt(posted.ID, 10)
	h.emptyMessages(t, messages+"?before="+id, "bob")
	h.emptyMessages(t, messages+"?after="+id, "bob")
	h.emptyMessages(t, messages+"?limit=2&before="+id, "bob")

	var page chatutil.ListMessagesResult
	h.expect(t, http.StatusOK, http.MethodGet, messages+"?limit=2", "bob", "", &page)
	if got := ids(page); len(got) != 1 || got[0] != posted.ID {
		t.Errorf("page = %v, want [%d]", got, posted.ID)
	}
}

// TestChatSearch_UnreadableRoomsStayHidden checks a room the caller cannot
// read contributes no message: a private channel they were never in, one
// they left, one they only manage, and one an admin is not a member of.
// The answer is 404, and the ciphertext stays out of the body. A cursor
// taken from another channel does not pull that channel's messages into
// one they can read.
func TestChatSearch_UnreadableRoomsStayHidden(t *testing.T) {
	h := newHarness(t)
	roomID, room := roomWithKey(t, h)
	messages := room + "/messages"
	var posted chatutil.Message
	h.expect(t, http.StatusCreated, http.MethodPost, messages, "bob", messageBody('s', 64, 1), &posted)
	secret := b64('s', 64)

	for _, as := range []string{"carol", "admin", "dave"} {
		h.refuseMessages(t, messages, as, secret, http.StatusNotFound)
	}
	h.refuseMessages(t, messages, "stranger", secret, http.StatusUnauthorized)

	h.expect(t, http.StatusOK, http.MethodPut, room+"/members", "bob", userBody(h.users["dave"], "manage_members"), nil)
	h.refuseMessages(t, messages, "dave", secret, http.StatusNotFound)

	// carol can read and was never granted the key. She still receives the
	// ciphertext. Keeping a message she cannot open out of the search
	// results is the device's job (#2429).
	h.expect(t, http.StatusOK, http.MethodPut, room+"/members", "bob", userBody(h.users["carol"], viewer...), nil)
	var page chatutil.ListMessagesResult
	h.expect(t, http.StatusOK, http.MethodGet, messages+"?after=0", "carol", "", &page)
	if got := ids(page); len(got) != 1 || got[0] != posted.ID || !bytes.Equal(page.Messages[0].Ciphertext, bytes.Repeat([]byte{'s'}, 64)) {
		t.Fatalf("carol reads %+v, want the stored ciphertext", page.Messages)
	}
	if page.Messages[0].ChannelID != roomID {
		t.Errorf("message channel = %d, want %d", page.Messages[0].ChannelID, roomID)
	}

	h.expect(t, http.StatusOK, http.MethodDelete, room+"/members", "carol", userBody(h.users["carol"]), nil)
	h.refuseMessages(t, messages, "carol", secret, http.StatusNotFound)

	h.expect(t, http.StatusOK, http.MethodGet, messages+"?after=0", "bob", "", &page)
	if got := ids(page); len(got) != 1 || got[0] != posted.ID {
		t.Errorf("bob reads %v after carol left, want [%d]", got, posted.ID)
	}

	var aside chatutil.Channel
	h.expect(t, http.StatusCreated, http.MethodPost, "/chat/channels", "bob", `{"name":"aside"}`, &aside)
	asidePath := "/chat/channels/" + strconv.FormatInt(aside.ID, 10)
	h.expect(t, http.StatusOK, http.MethodPost, asidePath+"/keys", "bob", createKeyBody('b', aside.ID, 1, h.users["bob"]), nil)
	var hidden chatutil.Message
	h.expect(t, http.StatusCreated, http.MethodPost, asidePath+"/messages", "bob", messageBody('h', 64, 1), &hidden)
	hiddenSecret := b64('h', 64)
	for _, as := range []string{"carol", "dave", "admin"} {
		h.refuseMessages(t, asidePath+"/messages", as, hiddenSecret, http.StatusNotFound)
	}

	code, body := h.do(t, http.MethodGet, messages+"?after=0", "bob", "")
	if code != http.StatusOK || strings.Contains(body, hiddenSecret) {
		t.Fatalf("bob's room = %d %s, want 200 without the other channel's ciphertext", code, body)
	}
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		t.Fatal(err)
	}
	for _, m := range page.Messages {
		if m.ID == hidden.ID || m.ChannelID != roomID {
			t.Errorf("room page includes %+v", m)
		}
	}

	var listed chatutil.ListChannelsResult
	h.expect(t, http.StatusOK, http.MethodGet, "/chat/channels", "carol", "", &listed)
	var generalID int64
	for _, channel := range listed.Channels {
		if channel.IsDefault {
			generalID = channel.ID
		}
	}
	if generalID == 0 {
		t.Fatal("carol has no general channel")
	}
	general := "/chat/channels/" + strconv.FormatInt(generalID, 10) + "/messages?limit=2&before=" + strconv.FormatInt(hidden.ID, 10)
	if got := h.emptyMessages(t, general, "carol"); strings.Contains(got, hiddenSecret) {
		t.Fatalf("general page before another channel's message = %s", got)
	}
}

// TestChatSearch_PlaintextQuery records the #3479 scenarios that have no
// route: a plaintext query, a pending send, and a lock flag. Search runs on
// the device (#2429).
func TestChatSearch_PlaintextQuery(t *testing.T) {
	t.Skip("gap #3540: GET /chat/channels/{id}/messages has no query, pending-send, or lock parameter. Do not add a server search to let this test run")
}
