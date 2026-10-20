package v0_chat_test

import (
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"slices"
	"strconv"
	"testing"

	"github.com/autobutler-org/quark/pkg/util/chatutil"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
	"github.com/autobutler-org/quark/pkg/util/ratelimitutil"
)

// The server half of the chat badges (#3478). A mention lives in ciphertext
// the Quark cannot read, so the app counts mentions itself; what it leans on
// here is that unreadCount names exactly the messages it has to scan: the
// newest unreadCount messages someone else wrote and nobody deleted, however
// many pages back they reach, in every channel at once.

// numberedBody is a post whose nonce carries n, so a channel can hold more
// messages than messageBody has fill bytes without a replay conflict.
func numberedBody(n int) string {
	ciphertext := make([]byte, 64)
	binary.BigEndian.PutUint64(ciphertext, uint64(n))
	return `{"keyVersion":1,"ciphertext":"` + base64.StdEncoding.EncodeToString(ciphertext) + `"}`
}

// badgeRoom has bob create the channel name with key version 1 and carol in
// it as a member, and returns its id and path.
func (h harness) badgeRoom(t *testing.T, name string) (int64, string) {
	t.Helper()
	h.expect(t, http.StatusOK, http.MethodPut, "/chat/keys/me", "bob", keysBody('b', false), nil)
	var room chatutil.Channel
	h.expect(t, http.StatusCreated, http.MethodPost, "/chat/channels", "bob", `{"name":"`+name+`"}`, &room)
	path := "/chat/channels/" + strconv.FormatInt(room.ID, 10)
	h.expect(t, http.StatusOK, http.MethodPost, path+"/keys", "bob", createKeyBody('b', room.ID, 1, h.users["bob"]), nil)
	h.expect(t, http.StatusOK, http.MethodPut, path+"/members", "bob", userBody(h.users["carol"], member...), nil)
	return room.ID, path
}

// poster posts numbered messages, each nonce new across the whole test.
type poster struct {
	h    harness
	next int
}

func (p *poster) post(t *testing.T, path, as string) int64 {
	t.Helper()
	p.next++
	var m chatutil.Message
	p.h.expect(t, http.StatusCreated, http.MethodPost, path+"/messages", as, numberedBody(p.next), &m)
	return m.ID
}

// unreadCounts is every channel's unreadCount in the account's channel list.
func (h harness) unreadCounts(t *testing.T, as string) map[int64]int64 {
	t.Helper()
	var result chatutil.ListChannelsResult
	h.expect(t, http.StatusOK, http.MethodGet, "/chat/channels", as, "", &result)
	counts := map[int64]int64{}
	for _, channel := range result.Channels {
		counts[channel.ID] = channel.UnreadCount
	}
	return counts
}

// countsBy reads a message as unread for as: someone else wrote it and
// nobody deleted it.
func countsBy(h harness, as string, m chatutil.Message) bool {
	return m.AuthorID != h.users[as] && m.DeletedAt == nil
}

// unreadSince pages forward from the read marker, the way a client that
// knows it would, and returns the ids that count as unread for as.
func (h harness) unreadSince(t *testing.T, path, as string, marker int64) []int64 {
	t.Helper()
	var out []int64
	for cursor := marker; ; {
		var page chatutil.ListMessagesResult
		h.expect(t, http.StatusOK, http.MethodGet, path+"/messages?after="+strconv.FormatInt(cursor, 10)+"&limit="+strconv.Itoa(chatutil.MaxMessagesPage), as, "", &page)
		if len(page.Messages) == 0 {
			return out
		}
		for _, m := range page.Messages {
			if countsBy(h, as, m) {
				out = append(out, m.ID)
			}
		}
		cursor = page.Messages[len(page.Messages)-1].ID
	}
}

// unreadFromNewest is the scan a client can make knowing only unreadCount:
// page back from the newest message until it has seen that many that count,
// and return them oldest first with the pages it took.
func (h harness) unreadFromNewest(t *testing.T, path, as string, count int64) (unread []int64, pages int) {
	t.Helper()
	for before := int64(0); int64(len(unread)) < count; pages++ {
		query := path + "/messages?limit=" + strconv.Itoa(chatutil.MaxMessagesPage)
		if before != 0 {
			query += "&before=" + strconv.FormatInt(before, 10)
		}
		var page chatutil.ListMessagesResult
		h.expect(t, http.StatusOK, http.MethodGet, query, as, "", &page)
		if len(page.Messages) == 0 {
			t.Fatalf("history ran out after %d of %d unread", len(unread), count)
		}
		for _, m := range slices.Backward(page.Messages) {
			if countsBy(h, as, m) && int64(len(unread)) < count {
				unread = append(unread, m.ID)
			}
		}
		before = page.Messages[0].ID
	}
	slices.Reverse(unread)
	return unread, pages
}

// readMarkers returns the chat_read_marker_changed events since the last
// drain.
func (h harness) readMarkers() []eventbus.ChatReadMarkerChanged {
	var heard []eventbus.ChatReadMarkerChanged
	for _, evt := range h.drain() {
		if data, ok := evt.Data.(eventbus.ChatReadMarkerChanged); ok && evt.Kind == eventbus.EventChatReadMarkerChanged {
			heard = append(heard, data)
		}
	}
	return heard
}

// TestChatUnread_DeepPageMention has bob mention carol in a channel she never
// opened and then bury it under more than a page of messages: the mention is
// message MaxMessagesPage+5 from the newest. The count reaches it, a scan back
// from the newest finds the same set a scan from the marker does, reading to
// the end clears it on every session, and one more message counts exactly 1.
func TestChatUnread_DeepPageMention(t *testing.T) {
	h := newHarness(t)
	h.deps.WithChatRateLimiter(ratelimitutil.NewWithRate(0, 1000))
	roomID, room := h.badgeRoom(t, "deep")
	p := &poster{h: h}

	mention := p.post(t, room, "bob")
	newest := mention
	for range chatutil.MaxMessagesPage + 4 {
		newest = p.post(t, room, "bob")
	}
	const total = chatutil.MaxMessagesPage + 5

	if got := h.unreadCounts(t, "carol")[roomID]; got != total {
		t.Fatalf("carol unreadCount = %d, want %d", got, total)
	}
	var first chatutil.ListMessagesResult
	h.expect(t, http.StatusOK, http.MethodGet, room+"/messages?limit="+strconv.Itoa(chatutil.MaxMessagesPage), "carol", "", &first)
	if slices.Contains(ids(first), mention) {
		t.Fatalf("the newest page holds the mention; it should be a page back")
	}
	back, pages := h.unreadFromNewest(t, room, "carol", total)
	if pages != 2 || back[0] != mention {
		t.Errorf("scan from the newest took %d pages and starts at %d, want 2 pages reaching the mention %d", pages, back[0], mention)
	}
	if forward := h.unreadSince(t, room, "carol", 0); !slices.Equal(back, forward) {
		t.Errorf("scan from the newest = %d ids, from the marker = %d ids; they should be the same set", len(back), len(forward))
	}

	h.drain()
	var read chatutil.MarkReadResult
	h.expect(t, http.StatusOK, http.MethodPut, room+"/read", "carol", readBody(newest), &read)
	if read.UnreadCount != 0 || h.unreadCounts(t, "carol")[roomID] != 0 {
		t.Errorf("reading to the end left %+v", read)
	}
	want := eventbus.ChatReadMarkerChanged{ChannelID: roomID, LastReadMessageID: newest, UnreadCount: 0, UserID: h.users["carol"]}
	if heard := h.readMarkers(); len(heard) != 1 || heard[0] != want {
		t.Errorf("carol's other sessions heard %+v, want %+v", heard, want)
	}

	next := p.post(t, room, "bob")
	if got := h.unreadCounts(t, "carol")[roomID]; got != 1 {
		t.Errorf("one message after the marker counts %d, want 1", got)
	}
	if got := h.unreadSince(t, room, "carol", newest); !slices.Equal(got, []int64{next}) {
		t.Errorf("unread after the marker = %v, want [%d]", got, next)
	}
}

// TestChatUnread_ConversationsAgree spreads unread over three channels, A=3,
// B=0 and C=5, with carol's own messages mixed in, and checks every
// channel's count, and so their sum, against the messages API. Reading C
// clears C alone, reading A halfway leaves what is after the marker, and a new
// message re-increments only its own channel by 1.
func TestChatUnread_ConversationsAgree(t *testing.T) {
	h := newHarness(t)
	p := &poster{h: h}
	aID, a := h.badgeRoom(t, "alpha")
	bID, b := h.badgeRoom(t, "beta")
	cID, c := h.badgeRoom(t, "gamma")
	paths := map[int64]string{aID: a, bID: b, cID: c}

	var aPosts []int64
	for range 3 {
		aPosts = append(aPosts, p.post(t, a, "bob"))
	}
	for range 2 {
		p.post(t, b, "carol")
	}
	var cNewest int64
	for i := range 5 {
		p.post(t, c, "bob")
		if i%2 == 0 {
			cNewest = p.post(t, c, "carol")
		}
	}

	check := func(step string, want map[int64]int64) {
		t.Helper()
		counts := h.unreadCounts(t, "carol")
		var sum, wantSum int64
		for id, path := range paths {
			if counts[id] != want[id] {
				t.Errorf("%s: channel %d counts %d, want %d", step, id, counts[id], want[id])
			}
			sum += counts[id]
			wantSum += want[id]
			back, _ := h.unreadFromNewest(t, path, "carol", counts[id])
			if int64(len(back)) != counts[id] {
				t.Errorf("%s: channel %d counts %d but the scan back finds %v", step, id, counts[id], back)
			}
		}
		if sum != wantSum {
			t.Errorf("%s: the counts sum to %d, want %d", step, sum, wantSum)
		}
	}
	check("before reading", map[int64]int64{aID: 3, bID: 0, cID: 5})
	for id, path := range paths {
		if got, want := h.unreadSince(t, path, "carol", 0), h.unreadCounts(t, "carol")[id]; int64(len(got)) != want {
			t.Errorf("channel %d: %d unread from the beginning, the list counts %d", id, len(got), want)
		}
	}
	if bobs := h.unreadCounts(t, "bob"); bobs[aID] != 0 || bobs[bID] != 2 || bobs[cID] != 3 {
		t.Errorf("bob counts %v, want only carol's messages: A=0 B=2 C=3", bobs)
	}

	// Carol's own message is the newest in C, and a marker on it is as good
	// as any.
	var read chatutil.MarkReadResult
	h.expect(t, http.StatusOK, http.MethodPut, c+"/read", "carol", readBody(cNewest), &read)
	if read.UnreadCount != 0 {
		t.Errorf("reading C to carol's own newest message left %d", read.UnreadCount)
	}
	check("C read", map[int64]int64{aID: 3, bID: 0, cID: 0})

	h.expect(t, http.StatusOK, http.MethodPut, a+"/read", "carol", readBody(aPosts[1]), &read)
	if read.UnreadCount != 1 {
		t.Errorf("reading A halfway answers %d unread, want 1", read.UnreadCount)
	}
	if got := h.unreadSince(t, a, "carol", aPosts[1]); !slices.Equal(got, aPosts[2:]) {
		t.Errorf("A after the marker = %v, want %v", got, aPosts[2:])
	}
	check("A half read", map[int64]int64{aID: 1, bID: 0, cID: 0})

	p.post(t, c, "bob")
	p.post(t, b, "carol")
	check("one more in C", map[int64]int64{aID: 1, bID: 0, cID: 1})
}

// TestChatUnread_OwnMessagesNeverCount has carol write after her marker,
// before and after bob does: the count is bob's message alone, and the
// response to reading, the event and the list all agree on it.
func TestChatUnread_OwnMessagesNeverCount(t *testing.T) {
	h := newHarness(t)
	p := &poster{h: h}
	roomID, room := h.badgeRoom(t, "own")

	marker := p.post(t, room, "bob")
	h.expect(t, http.StatusOK, http.MethodPut, room+"/read", "carol", readBody(marker), nil)
	for range 3 {
		p.post(t, room, "carol")
	}
	bobs := p.post(t, room, "bob")
	carols := p.post(t, room, "carol")

	if got := h.unreadCounts(t, "carol")[roomID]; got != 1 {
		t.Errorf("carol unreadCount = %d, want 1: only bob's message", got)
	}
	if got := h.unreadSince(t, room, "carol", marker); !slices.Equal(got, []int64{bobs}) {
		t.Errorf("unread after the marker = %v, want [%d]", got, bobs)
	}

	h.drain()
	var read chatutil.MarkReadResult
	h.expect(t, http.StatusOK, http.MethodPut, room+"/read", "carol", readBody(carols), &read)
	heard := h.readMarkers()
	if read.UnreadCount != 0 || len(heard) != 1 || heard[0].UnreadCount != 0 || h.unreadCounts(t, "carol")[roomID] != 0 {
		t.Errorf("reading to carol's own message: response %+v, events %+v", read, heard)
	}
}

// TestChatUnread_DeletedMention has bob delete the message that mentioned
// carol before she opens the channel: the count drops with it, the scan
// back skips its tombstone, and a deletion after she has read changes
// nothing.
func TestChatUnread_DeletedMention(t *testing.T) {
	h := newHarness(t)
	p := &poster{h: h}
	roomID, room := h.badgeRoom(t, "deleted")

	plain := p.post(t, room, "bob")
	mention := p.post(t, room, "bob")
	if got := h.unreadCounts(t, "carol")[roomID]; got != 2 {
		t.Fatalf("carol unreadCount = %d, want 2", got)
	}
	h.expect(t, http.StatusOK, http.MethodDelete, "/chat/messages/"+strconv.FormatInt(mention, 10), "bob", "", nil)

	if got := h.unreadCounts(t, "carol")[roomID]; got != 1 {
		t.Errorf("after the mention is deleted carol counts %d, want 1", got)
	}
	if back, _ := h.unreadFromNewest(t, room, "carol", 1); !slices.Equal(back, []int64{plain}) {
		t.Errorf("scan back = %v, want [%d] past the tombstone", back, plain)
	}

	h.expect(t, http.StatusOK, http.MethodPut, room+"/read", "carol", readBody(mention), nil)
	h.expect(t, http.StatusOK, http.MethodDelete, "/chat/messages/"+strconv.FormatInt(plain, 10), "bob", "", nil)
	if got := h.unreadCounts(t, "carol")[roomID]; got != 0 {
		t.Errorf("deleting a read message left carol counting %d", got)
	}
}

// TestChatUnread_EditedMention is scenario 5's edit half: bob edits the only
// unread mention out of his message before carol opens the channel, and the
// app's count has to drop while unreadCount stays 1. The API has no way to
// edit a message yet (#2426), so there is nothing to pin.
func TestChatUnread_EditedMention(t *testing.T) {
	t.Skip("gap #3539: no endpoint edits a chat message (#2426), so an edit that removes a mention cannot be pinned")
}
