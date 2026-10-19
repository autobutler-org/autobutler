package v0_chat_test

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/autobutler-org/quark/pkg/util/accessutil"
	"github.com/autobutler-org/quark/pkg/util/chatutil"
	"github.com/autobutler-org/quark/pkg/util/eventbus"
)

// Sessions below are the two devices of one account plus another account from
// #3482. Each session has its own bus subscription, filtered with
// accessutil.FilterEvent the way GET /api/v0/events filters a socket, and
// builds a view from those events. Catch-up re-reads the loaded window with
// GET /chat/channels/{id}/messages?after=, which is what a client does after a
// reconnect or a resync. The view then matches a fresh server page: message
// ids and order, ciphertext, reactions, and tombstones.
//
// Flutter is out of scope here: two widget controllers, the drawer badge,
// mention badges (a mention is ciphertext; the server never sees it,
// JN-CHAT-020), and the offline send queue.

// convergeBarrier is published once a test wants every connected session to
// finish applying what was already published. Its path keeps it distinct from
// a real resync, which has an empty path.
const convergeBarrier = "converge-barrier"

// session is one signed-in client. Two sessions with the same as are two
// devices of one account.
type session struct {
	h      harness
	name   string
	as     string
	userID int64
	access accessutil.Access

	events    <-chan eventbus.Event
	unsub     func()
	connected bool
	lossy     bool
	gen       int

	msgs       map[int64]chatutil.Message
	created    []int64
	channelHit []int64
	readEvents []eventbus.ChatReadMarkerChanged
	// lastRead is the marker learned from chat_read_marker_changed. It stays
	// 0 when this session has not seen one.
	lastRead int64
	// windowFrom is the oldest message id rendered before a resync. A tail
	// that arrives after the resync must not move it, or catch-up would
	// start in the middle of the gap.
	windowFrom int64
	frozen     bool
	resynced   bool
}

func openSession(t *testing.T, h harness, as, name string, lossy bool) *session {
	t.Helper()
	// Chat filtering reads only the principal. A nil database makes Load
	// skip file grants, which these events do not consult.
	loaded, err := accessutil.Load(accessutil.LoadParams{
		Ctx:       context.Background(),
		Principal: accessutil.Principal{UserID: h.users[as]},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := &session{
		h: h, name: name, as: as, userID: h.users[as], access: loaded.Access,
		msgs: map[int64]chatutil.Message{}, lossy: lossy,
	}
	s.subscribe(t, name)
	return s
}

func (s *session) subscribe(t *testing.T, name string) {
	t.Helper()
	id := "converge-" + s.as + "-" + name
	var events <-chan eventbus.Event
	var unsub func()
	if s.lossy {
		events, unsub = s.h.deps.EventBus().SubscribeLossy(id)
	} else {
		events, unsub = s.h.deps.EventBus().Subscribe(id)
	}
	s.events = events
	s.unsub = unsub
	s.connected = true
	t.Cleanup(unsub)
}

func (s *session) disconnect() {
	s.unsub()
	s.connected = false
}

func (s *session) reconnect(t *testing.T) {
	t.Helper()
	s.gen++
	s.lossy = false
	s.frozen = false
	s.subscribe(t, s.name+"-"+strconv.Itoa(s.gen))
}

// deliver blocks until every connected session has applied the events
// published before the call. Lossy sessions are not waited on: a full buffer
// replaces its backlog with a resync and would drop the barrier.
func deliver(t *testing.T, h harness, sessions ...*session) {
	t.Helper()
	h.deps.EventBus().Publish(eventbus.Event{Kind: eventbus.EventResync, Path: convergeBarrier})
	for _, s := range sessions {
		if s.connected && !s.lossy {
			s.readUntilBarrier(t)
		}
	}
}

func (s *session) readUntilBarrier(t *testing.T) {
	t.Helper()
	for evt := range s.events {
		if evt.Kind == eventbus.EventResync && evt.Path == convergeBarrier {
			return
		}
		s.apply(t, evt)
	}
	t.Fatal("event stream closed before the barrier")
}

// drainLossy reads whatever a SubscribeLossy buffer holds now. Publish has
// already returned, and a lossy deliver writes the buffer before it returns,
// so a non-blocking read sees the resync and any tail.
func (s *session) drainLossy(t *testing.T) {
	t.Helper()
	for {
		select {
		case evt, ok := <-s.events:
			if !ok {
				t.Fatal("lossy stream closed")
			}
			if evt.Kind == eventbus.EventResync && evt.Path == convergeBarrier {
				continue
			}
			s.apply(t, evt)
		default:
			return
		}
	}
}

func (s *session) apply(t *testing.T, evt eventbus.Event) {
	t.Helper()
	if !accessutil.FilterEvent(accessutil.FilterEventParams{Access: s.access, Event: evt}).Deliver {
		return
	}
	switch evt.Kind {
	case eventbus.EventResync:
		s.resynced = true
		s.frozen = true
	case eventbus.EventChatMessageCreated:
		data, ok := evt.Data.(eventbus.ChatMessageChanged)
		if !ok {
			t.Fatalf("chat_message_created data %T", evt.Data)
		}
		msg, ok := data.Message.(chatutil.Message)
		if !ok {
			t.Fatalf("chat_message_created message %T", data.Message)
		}
		s.noteWindow(msg.ID)
		s.putEvent(msg)
		s.created = append(s.created, msg.ID)
	case eventbus.EventChatMessageDeleted:
		data, ok := evt.Data.(eventbus.ChatMessageChanged)
		if !ok {
			t.Fatalf("chat_message_deleted data %T", evt.Data)
		}
		s.noteWindow(data.MessageID)
		s.tombstone(data)
	case eventbus.EventChatReactionChanged:
		data, ok := evt.Data.(eventbus.ChatReactionChanged)
		if !ok {
			t.Fatalf("chat_reaction_changed data %T", evt.Data)
		}
		switch reaction := data.Reaction.(type) {
		case nil:
			s.removeReaction(data.MessageID, data.ReactionID)
		case chatutil.Reaction:
			s.addReaction(reaction)
		default:
			t.Fatalf("reaction %T", data.Reaction)
		}
	case eventbus.EventChatReadMarkerChanged:
		data, ok := evt.Data.(eventbus.ChatReadMarkerChanged)
		if !ok {
			t.Fatalf("chat_read_marker_changed data %T", evt.Data)
		}
		s.readEvents = append(s.readEvents, data)
		if data.LastReadMessageID > s.lastRead {
			s.lastRead = data.LastReadMessageID
		}
	case eventbus.EventChatChannelChanged:
		data, ok := evt.Data.(eventbus.ChatChannelChanged)
		if !ok {
			t.Fatalf("chat_channel_changed data %T", evt.Data)
		}
		s.channelHit = append(s.channelHit, data.ChannelID)
	}
}

func (s *session) noteWindow(id int64) {
	if s.frozen {
		return
	}
	if s.windowFrom == 0 || id < s.windowFrom {
		s.windowFrom = id
	}
}

// putEvent keeps one row per id and will not let a late create bring a
// tombstone back.
func (s *session) putEvent(msg chatutil.Message) {
	if prev, ok := s.msgs[msg.ID]; ok && prev.DeletedAt != nil && msg.DeletedAt == nil {
		return
	}
	s.msgs[msg.ID] = msg
}

func (s *session) putServer(msg chatutil.Message) {
	s.msgs[msg.ID] = msg
}

func (s *session) tombstone(data eventbus.ChatMessageChanged) {
	msg := s.msgs[data.MessageID]
	msg.ID = data.MessageID
	msg.ChannelID = data.ChannelID
	msg.Ciphertext = nil
	msg.Reactions = nil
	if msg.DeletedAt == nil {
		// The delete event carries no tombstone. Any non-nil marker is enough
		// until catch-up replaces it with the server's deletedAt.
		stamp := time.Unix(1, 0).UTC()
		msg.DeletedAt = &stamp
	}
	s.msgs[data.MessageID] = msg
}

func (s *session) addReaction(reaction chatutil.Reaction) {
	msg, ok := s.msgs[reaction.MessageID]
	if !ok || msg.DeletedAt != nil {
		return
	}
	for i, held := range msg.Reactions {
		if held.ID == reaction.ID {
			msg.Reactions[i] = reaction
			s.msgs[reaction.MessageID] = msg
			return
		}
	}
	msg.Reactions = append(msg.Reactions, reaction)
	s.msgs[reaction.MessageID] = msg
}

func (s *session) removeReaction(messageID, reactionID int64) {
	msg, ok := s.msgs[messageID]
	if !ok {
		return
	}
	kept := make([]chatutil.Reaction, 0, len(msg.Reactions))
	for _, held := range msg.Reactions {
		if held.ID != reactionID {
			kept = append(kept, held)
		}
	}
	if len(kept) == 0 {
		kept = nil
	}
	msg.Reactions = kept
	s.msgs[messageID] = msg
}

// catchUp re-reads from the oldest id rendered before a gap, then forward,
// and lets the server row win. A session that rendered nothing, or that only
// saw a resync tail, starts at the beginning so the tail cannot hide the gap.
func (s *session) catchUp(t *testing.T, messages string) {
	t.Helper()
	after := int64(0)
	if s.windowFrom > 0 {
		after = s.windowFrom - 1
	}
	for {
		var page chatutil.ListMessagesResult
		path := messages + "?after=" + strconv.FormatInt(after, 10) + "&limit=200"
		s.h.expect(t, http.StatusOK, http.MethodGet, path, s.as, "", &page)
		if len(page.Messages) == 0 {
			return
		}
		for _, msg := range page.Messages {
			s.putServer(msg)
			s.noteWindow(msg.ID)
		}
		next := page.Messages[len(page.Messages)-1].ID
		if next <= after {
			t.Fatalf("catch-up page did not advance past %d", after)
		}
		after = next
	}
}

func (s *session) messages() []chatutil.Message {
	ids := make([]int64, 0, len(s.msgs))
	for id := range s.msgs {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	out := make([]chatutil.Message, len(ids))
	for i, id := range ids {
		out[i] = s.msgs[id]
	}
	return out
}

// unread is the server's count, worked out from the view: messages after the
// marker that someone else wrote and nobody deleted.
func (s *session) unread(channelID int64) int64 {
	var n int64
	for _, msg := range s.messages() {
		if msg.ChannelID != channelID || msg.DeletedAt != nil || msg.AuthorID == s.userID || msg.ID <= s.lastRead {
			continue
		}
		n++
	}
	return n
}

func (s *session) takeChannel(id int64) int {
	n := 0
	kept := make([]int64, 0, len(s.channelHit))
	for _, hit := range s.channelHit {
		if hit == id {
			n++
		} else {
			kept = append(kept, hit)
		}
	}
	s.channelHit = kept
	return n
}

func (s *session) assertCreatedOnce(t *testing.T, want []int64) {
	t.Helper()
	got := slices.Clone(s.created)
	slices.Sort(got)
	seen := map[int64]int{}
	for _, id := range s.created {
		seen[id]++
		if seen[id] > 1 {
			t.Errorf("session %s heard message %d %d times", s.name, id, seen[id])
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("session %s heard %v, want each of %v once", s.name, s.created, want)
	}
}

func allMessages(t *testing.T, h harness, messages, as string) []chatutil.Message {
	t.Helper()
	var all []chatutil.Message
	for after := int64(0); ; {
		var page chatutil.ListMessagesResult
		path := messages + "?after=" + strconv.FormatInt(after, 10) + "&limit=200"
		h.expect(t, http.StatusOK, http.MethodGet, path, as, "", &page)
		if len(page.Messages) == 0 {
			return all
		}
		all = append(all, page.Messages...)
		next := all[len(all)-1].ID
		if next <= after {
			t.Fatalf("server page did not advance past %d", after)
		}
		after = next
	}
}

func postMsg(t *testing.T, h harness, messages, as string, fill byte) chatutil.Message {
	t.Helper()
	var msg chatutil.Message
	h.expect(t, http.StatusCreated, http.MethodPost, messages, as, messageBody(fill, 64, 1), &msg)
	return msg
}

func addReact(t *testing.T, h harness, messageID int64, as string, fill byte) chatutil.Reaction {
	t.Helper()
	var reaction chatutil.Reaction
	path := "/chat/messages/" + strconv.FormatInt(messageID, 10) + "/reactions"
	h.expect(t, http.StatusCreated, http.MethodPost, path, as, messageBody(fill, 48, 1), &reaction)
	return reaction
}

func sameMessage(got, want chatutil.Message, strictDeletedAt bool) bool {
	if got.ID != want.ID || got.ChannelID != want.ChannelID || got.AuthorID != want.AuthorID || got.KeyVersion != want.KeyVersion {
		return false
	}
	if !bytes.Equal(got.Ciphertext, want.Ciphertext) {
		return false
	}
	if (got.DeletedAt == nil) != (want.DeletedAt == nil) {
		return false
	}
	if strictDeletedAt && got.DeletedAt != nil && !got.DeletedAt.Equal(*want.DeletedAt) {
		return false
	}
	if len(got.Reactions) != len(want.Reactions) {
		return false
	}
	gr, wr := slices.Clone(got.Reactions), slices.Clone(want.Reactions)
	slices.SortFunc(gr, func(a, b chatutil.Reaction) int { return cmp.Compare(a.ID, b.ID) })
	slices.SortFunc(wr, func(a, b chatutil.Reaction) int { return cmp.Compare(a.ID, b.ID) })
	for i := range gr {
		if gr[i].ID != wr[i].ID || gr[i].MessageID != wr[i].MessageID || gr[i].UserID != wr[i].UserID ||
			gr[i].KeyVersion != wr[i].KeyVersion || !bytes.Equal(gr[i].Ciphertext, wr[i].Ciphertext) {
			return false
		}
	}
	return true
}

func requirePage(t *testing.T, h harness, messages string, strict bool, sessions ...*session) {
	t.Helper()
	server := allMessages(t, h, messages, sessions[0].as)
	for _, other := range sessions[1:] {
		if got := allMessages(t, h, messages, other.as); !sameMessages(server, got, true) {
			t.Fatalf("%s reads %s, %s reads %s", sessions[0].as, describe(server), other.as, describe(got))
		}
	}
	for _, s := range sessions {
		if got := s.messages(); !sameMessages(got, server, strict) {
			t.Errorf("session %s view %s, server %s", s.name, describe(got), describe(server))
		}
	}
}

func sameMessages(got, want []chatutil.Message, strict bool) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if !sameMessage(got[i], want[i], strict) {
			return false
		}
	}
	return true
}

func describe(msgs []chatutil.Message) string {
	buf := bytes.NewBufferString("[")
	for i, msg := range msgs {
		if i > 0 {
			buf.WriteByte(' ')
		}
		reactions := make([]int64, len(msg.Reactions))
		for j, reaction := range msg.Reactions {
			reactions[j] = reaction.ID
		}
		buf.WriteString(strconv.FormatInt(msg.ID, 10))
		buf.WriteString("{author=")
		buf.WriteString(strconv.FormatInt(msg.AuthorID, 10))
		if msg.DeletedAt != nil {
			buf.WriteString(" deleted")
		}
		if len(msg.Ciphertext) > 0 {
			buf.WriteString(" bytes=")
			buf.WriteString(strconv.Itoa(len(msg.Ciphertext)))
		}
		if len(reactions) > 0 {
			buf.WriteString(" reactions=")
			for j, id := range reactions {
				if j > 0 {
					buf.WriteByte(',')
				}
				buf.WriteString(strconv.FormatInt(id, 10))
			}
		}
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
	return buf.String()
}

func requireUnread(t *testing.T, h harness, channelID int64, sessions ...*session) {
	t.Helper()
	for _, s := range sessions {
		want := h.unreadCount(t, s.as, channelID)
		if got := s.unread(channelID); got != want {
			t.Errorf("session %s unread %d, server %d for %s", s.name, got, want, s.as)
		}
	}
}

func messageIDs(msgs []chatutil.Message) []int64 {
	out := make([]int64, len(msgs))
	for i, msg := range msgs {
		out[i] = msg.ID
	}
	return out
}

func memberUserIDs(t *testing.T, h harness, path, as string) []int64 {
	t.Helper()
	var result chatutil.ListMembersResult
	h.expect(t, http.StatusOK, http.MethodGet, path+"/members", as, "", &result)
	ids := make([]int64, 0, len(result.Members))
	for _, member := range result.Members {
		if member.UserID != 0 {
			ids = append(ids, member.UserID)
		}
	}
	slices.Sort(ids)
	return ids
}

func listedChannel(t *testing.T, h harness, as string, channelID int64) (chatutil.Channel, bool) {
	t.Helper()
	var result chatutil.ListChannelsResult
	h.expect(t, http.StatusOK, http.MethodGet, "/chat/channels", as, "", &result)
	for _, channel := range result.Channels {
		if channel.ID == channelID {
			return channel, true
		}
	}
	return chatutil.Channel{}, false
}

// TestChatConverge_Send has bob's two sessions and carol's one session read
// the same messages, in the server's order, each id once. bob's own sends do
// not become unread on his other session; carol's copy of those sends does.
func TestChatConverge_Send(t *testing.T) {
	h := newHarness(t)
	roomID, room := roomWithKey(t, h)
	messages := room + "/messages"
	h.expect(t, http.StatusOK, http.MethodPut, room+"/members", "bob", userBody(h.users["carol"], member...), nil)
	bob1 := openSession(t, h, "bob", "bob-1", false)
	bob2 := openSession(t, h, "bob", "bob-2", false)
	carol := openSession(t, h, "carol", "carol", false)

	first := postMsg(t, h, messages, "bob", 'a')
	second := postMsg(t, h, messages, "carol", 'b')
	third := postMsg(t, h, messages, "bob", 'c')
	deliver(t, h, bob1, bob2, carol)

	// Event delivery alone matches the stored rows. Own messages stay out of
	// bob's unread count on both of his sessions.
	requirePage(t, h, messages, false, bob1, bob2, carol)
	requireUnread(t, h, roomID, bob1, bob2, carol)
	if got := h.unreadCount(t, "bob", roomID); got != 1 {
		t.Errorf("bob unread = %d, want 1 (carol's message only)", got)
	}
	if got := h.unreadCount(t, "carol", roomID); got != 2 {
		t.Errorf("carol unread = %d, want 2 (bob's two messages)", got)
	}
	want := []int64{first.ID, second.ID, third.ID}
	bob1.assertCreatedOnce(t, want)
	bob2.assertCreatedOnce(t, want)
	carol.assertCreatedOnce(t, want)

	bob1.catchUp(t, messages)
	bob2.catchUp(t, messages)
	carol.catchUp(t, messages)
	requirePage(t, h, messages, true, bob1, bob2, carol)
	requireUnread(t, h, roomID, bob1, bob2, carol)
	if len(bob2.messages()) != len(want) || len(bob2.created) != len(want) {
		t.Errorf("catch-up changed bob-2 from %d messages heard once", len(want))
	}
}

// TestChatConverge_Read marks a channel read on one of carol's sessions and
// checks the other session hears the same marker, while bob's unread is left
// as it was.
func TestChatConverge_Read(t *testing.T) {
	h := newHarness(t)
	roomID, room := roomWithKey(t, h)
	messages := room + "/messages"
	h.expect(t, http.StatusOK, http.MethodPut, room+"/members", "bob", userBody(h.users["carol"], member...), nil)
	bob := openSession(t, h, "bob", "bob", false)
	carol1 := openSession(t, h, "carol", "carol-1", false)
	carol2 := openSession(t, h, "carol", "carol-2", false)

	posted := make([]chatutil.Message, 3)
	for i := range posted {
		posted[i] = postMsg(t, h, messages, "bob", byte('1'+i))
	}
	deliver(t, h, bob, carol1, carol2)
	if got := h.unreadCount(t, "carol", roomID); got != 3 {
		t.Fatalf("carol unread = %d, want 3", got)
	}
	if got := h.unreadCount(t, "bob", roomID); got != 0 {
		t.Fatalf("bob unread of his own messages = %d, want 0", got)
	}
	requireUnread(t, h, roomID, bob, carol1, carol2)

	var marked chatutil.MarkReadResult
	h.expect(t, http.StatusOK, http.MethodPut, room+"/read", "carol", readBody(posted[1].ID), &marked)
	deliver(t, h, bob, carol1, carol2)

	want := eventbus.ChatReadMarkerChanged{
		ChannelID: roomID, LastReadMessageID: posted[1].ID, UnreadCount: 1, UserID: h.users["carol"],
	}
	for _, s := range []*session{carol1, carol2} {
		if len(s.readEvents) != 1 || s.readEvents[0] != want || s.lastRead != posted[1].ID {
			t.Errorf("session %s marker events = %+v, want %+v", s.name, s.readEvents, want)
		}
	}
	if len(bob.readEvents) != 0 {
		t.Errorf("bob heard carol's marker: %+v", bob.readEvents)
	}
	if got := h.unreadCount(t, "bob", roomID); got != 0 {
		t.Errorf("bob unread after carol read = %d, want 0", got)
	}
	requireUnread(t, h, roomID, carol1, carol2)
	if carol2.unread(roomID) != marked.UnreadCount {
		t.Errorf("carol-2 unread %d, mark-read answered %d", carol2.unread(roomID), marked.UnreadCount)
	}

	// A later message moves the count and not the marker. carol's own send
	// does not, and reading to the end on carol-1 leaves bob's new unread.
	fourth := postMsg(t, h, messages, "bob", '4')
	deliver(t, h, bob, carol1, carol2)
	if carol2.lastRead != posted[1].ID || carol2.unread(roomID) != 2 {
		t.Errorf("after a new message, carol-2 marker %d unread %d, want marker %d unread 2", carol2.lastRead, carol2.unread(roomID), posted[1].ID)
	}
	hers := postMsg(t, h, messages, "carol", 'k')
	deliver(t, h, bob, carol1, carol2)
	if got := h.unreadCount(t, "bob", roomID); got != 1 {
		t.Fatalf("bob unread of carol's message = %d, want 1", got)
	}
	h.expect(t, http.StatusOK, http.MethodPut, room+"/read", "carol", readBody(hers.ID), nil)
	deliver(t, h, bob, carol1, carol2)
	if got := h.unreadCount(t, "carol", roomID); got != 0 || carol2.lastRead != hers.ID {
		t.Errorf("carol unread %d marker %d, want 0 at %d", got, carol2.lastRead, hers.ID)
	}
	if got := h.unreadCount(t, "bob", roomID); got != 1 {
		t.Errorf("bob unread after carol finished reading = %d, want 1", got)
	}
	requireUnread(t, h, roomID, bob, carol1, carol2)
	if fourth.ID <= posted[1].ID {
		t.Fatalf("message ids did not advance")
	}
}

// TestChatConverge_ReactionsAndDelete adds, removes and deletes while one of
// carol's sessions is connected and again while it is disconnected. Catch-up
// shows one copy of the surviving reaction and a tombstone where a message
// carol had already rendered was deleted.
func TestChatConverge_ReactionsAndDelete(t *testing.T) {
	h := newHarness(t)
	roomID, room := roomWithKey(t, h)
	messages := room + "/messages"
	h.expect(t, http.StatusOK, http.MethodPut, room+"/members", "bob", userBody(h.users["carol"], member...), nil)
	bob := openSession(t, h, "bob", "bob", false)
	carol1 := openSession(t, h, "carol", "carol-1", false)
	carol2 := openSession(t, h, "carol", "carol-2", false)

	live := postMsg(t, h, messages, "bob", 'm')
	deliver(t, h, bob, carol1, carol2)
	kept := addReact(t, h, live.ID, "carol", 'r')
	dropped := addReact(t, h, live.ID, "bob", 's')
	deliver(t, h, bob, carol1, carol2)
	h.expect(t, http.StatusOK, http.MethodDelete, "/chat/reactions/"+strconv.FormatInt(dropped.ID, 10), "bob", "", nil)
	deliver(t, h, bob, carol1, carol2)

	for _, s := range []*session{bob, carol1, carol2} {
		msg := s.msgs[live.ID]
		if len(msg.Reactions) != 1 || msg.Reactions[0].ID != kept.ID {
			t.Fatalf("session %s reactions = %+v, want only %d", s.name, msg.Reactions, kept.ID)
		}
	}
	carol1.catchUp(t, messages)
	carol2.catchUp(t, messages)
	bob.catchUp(t, messages)
	requirePage(t, h, messages, true, bob, carol1, carol2)
	if n := len(carol2.msgs[live.ID].Reactions); n != 1 {
		t.Fatalf("catch-up left carol-2 with %d reactions, want 1", n)
	}

	h.expect(t, http.StatusOK, http.MethodDelete, "/chat/messages/"+strconv.FormatInt(live.ID, 10), "bob", "", nil)
	deliver(t, h, bob, carol1, carol2)
	for _, s := range []*session{bob, carol1, carol2} {
		msg := s.msgs[live.ID]
		if msg.DeletedAt == nil || len(msg.Ciphertext) != 0 || len(msg.Reactions) != 0 {
			t.Errorf("session %s tombstone = %+v", s.name, msg)
		}
	}
	if got := h.unreadCount(t, "carol", roomID); got != 0 {
		t.Errorf("carol unread after the delete = %d, want 0", got)
	}
	requireUnread(t, h, roomID, carol1, carol2)

	// carol-2 has rendered seen and stay. She misses stay's reactions and
	// seen's deletion, plus a message she never saw. Reconnecting re-reads
	// the window: seen is a tombstone, stay keeps one reaction, missed
	// arrives once.
	seen := postMsg(t, h, messages, "bob", 'n')
	stay := postMsg(t, h, messages, "bob", 'p')
	deliver(t, h, bob, carol1, carol2)
	before := len(carol2.created)
	carol2.disconnect()
	removed := addReact(t, h, stay.ID, "carol", 't')
	surviving := addReact(t, h, stay.ID, "bob", 'u')
	h.expect(t, http.StatusOK, http.MethodDelete, "/chat/reactions/"+strconv.FormatInt(removed.ID, 10), "carol", "", nil)
	h.expect(t, http.StatusOK, http.MethodDelete, "/chat/messages/"+strconv.FormatInt(seen.ID, 10), "bob", "", nil)
	missed := postMsg(t, h, messages, "carol", 'v')
	deliver(t, h, bob, carol1)
	if len(carol2.created) != before {
		t.Fatalf("disconnected carol-2 heard %d new creates", len(carol2.created)-before)
	}
	carol2.reconnect(t)
	carol2.catchUp(t, messages)
	bob.catchUp(t, messages)
	carol1.catchUp(t, messages)
	requirePage(t, h, messages, true, bob, carol1, carol2)
	requireUnread(t, h, roomID, bob, carol1, carol2)
	if carol2.msgs[seen.ID].DeletedAt == nil || len(carol2.msgs[seen.ID].Ciphertext) != 0 || len(carol2.msgs[seen.ID].Reactions) != 0 {
		t.Errorf("seen message resurrected on carol-2: %+v", carol2.msgs[seen.ID])
	}
	if got := carol2.msgs[stay.ID].Reactions; len(got) != 1 || got[0].ID != surviving.ID {
		t.Errorf("stay reactions on carol-2 = %+v, want only %d", got, surviving.ID)
	}
	if carol2.msgs[missed.ID].ID != missed.ID || carol2.msgs[missed.ID].DeletedAt != nil {
		t.Errorf("missed message on carol-2 = %+v", carol2.msgs[missed.ID])
	}
	if slices.Contains(carol2.created, missed.ID) {
		t.Errorf("catch-up also delivered missed message %d as an event", missed.ID)
	}
}

// TestChatConverge_Offline is #3482's offline case: carol's second session
// drops off, bob sends five messages and deletes one, carol sends one from
// her first session, and the second session's catch-up is the server page
// once each.
func TestChatConverge_Offline(t *testing.T) {
	h := newHarness(t)
	roomID, room := roomWithKey(t, h)
	messages := room + "/messages"
	h.expect(t, http.StatusOK, http.MethodPut, room+"/members", "bob", userBody(h.users["carol"], member...), nil)
	bob := openSession(t, h, "bob", "bob", false)
	carol1 := openSession(t, h, "carol", "carol-1", false)
	carol2 := openSession(t, h, "carol", "carol-2", false)
	// A message both of carol's sessions have already rendered, so the
	// window catch-up has something to re-read.
	postMsg(t, h, messages, "bob", '0')
	deliver(t, h, bob, carol1, carol2)
	carol2.disconnect()

	sent := make([]chatutil.Message, 5)
	for i := range sent {
		sent[i] = postMsg(t, h, messages, "bob", byte('a'+i))
	}
	h.expect(t, http.StatusOK, http.MethodDelete, "/chat/messages/"+strconv.FormatInt(sent[2].ID, 10), "bob", "", nil)
	hers := postMsg(t, h, messages, "carol", 'z')
	deliver(t, h, bob, carol1)

	carol2.reconnect(t)
	carol2.catchUp(t, messages)
	carol1.catchUp(t, messages)
	bob.catchUp(t, messages)
	requirePage(t, h, messages, true, bob, carol1, carol2)
	requireUnread(t, h, roomID, bob, carol1, carol2)

	if carol2.msgs[sent[2].ID].DeletedAt == nil || len(carol2.msgs[sent[2].ID].Ciphertext) != 0 {
		t.Errorf("deleted message %d came back on carol-2", sent[2].ID)
	}
	if carol2.msgs[hers.ID].AuthorID != h.users["carol"] {
		t.Errorf("carol's message on carol-2 = %+v", carol2.msgs[hers.ID])
	}
	for _, msg := range sent {
		if slices.Contains(carol2.created, msg.ID) {
			t.Errorf("disconnected session heard bob's message %d", msg.ID)
		}
	}
	server := allMessages(t, h, messages, "carol")
	if got := messageIDs(carol2.messages()); !slices.Equal(got, messageIDs(server)) {
		t.Fatalf("carol-2 ids %v, server %v", got, messageIDs(server))
	}
	// Five new messages plus the seed, minus nothing: the deletion is a
	// tombstone, so the page still has 7 rows (seed, five, carol's).
	if len(server) != 7 {
		t.Fatalf("server page length %d, want 7", len(server))
	}
}

// TestChatConverge_Resync fills a lossy socket past its buffer, then catches
// up. The resync tail is a subset of the server page, and merging it does
// not duplicate or drop an id. The sessions that stayed caught up heard each
// id once.
func TestChatConverge_Resync(t *testing.T) {
	h := newHarness(t)
	_, room := roomWithKey(t, h)
	messages := room + "/messages"
	h.expect(t, http.StatusOK, http.MethodPut, room+"/members", "bob", userBody(h.users["carol"], member...), nil)
	bob := openSession(t, h, "bob", "bob", false)
	carol := openSession(t, h, "carol", "carol", false)
	behind := openSession(t, h, "bob", "bob-behind", true)

	const burst = 20
	posted := make([]chatutil.Message, burst)
	for i := range posted {
		posted[i] = postMsg(t, h, messages, "bob", byte('a'+i))
	}
	behind.drainLossy(t)
	if !behind.resynced {
		t.Fatal("lossy session did not resync after falling behind")
	}
	deliver(t, h, bob, carol)
	want := messageIDs(posted)
	bob.assertCreatedOnce(t, want)
	carol.assertCreatedOnce(t, want)
	for _, id := range behind.created {
		if !slices.Contains(want, id) {
			t.Errorf("resync tail delivered %d, which was not posted", id)
		}
	}
	if len(behind.created) == 0 || len(behind.created) >= burst {
		t.Fatalf("resync tail has %d ids, want a non-empty subset of %d", len(behind.created), burst)
	}

	behind.catchUp(t, messages)
	bob.catchUp(t, messages)
	carol.catchUp(t, messages)
	requirePage(t, h, messages, true, bob, carol, behind)
	if got := messageIDs(behind.messages()); !slices.Equal(got, want) {
		t.Fatalf("behind ids %v, want %v", got, want)
	}
}

// TestChatConverge_ChannelState creates, renames, shares, leaves and removes
// a channel on one session and checks the account's other session is told and
// reads the same name, topic and members again. Removal takes the channel away
// from both of the removed account's sessions.
func TestChatConverge_ChannelState(t *testing.T) {
	h := newHarness(t)
	h.expect(t, http.StatusOK, http.MethodPut, "/chat/keys/me", "bob", keysBody('b', false), nil)
	bob1 := openSession(t, h, "bob", "bob-1", false)
	bob2 := openSession(t, h, "bob", "bob-2", false)
	carol1 := openSession(t, h, "carol", "carol-1", false)
	carol2 := openSession(t, h, "carol", "carol-2", false)

	var channel chatutil.Channel
	h.expect(t, http.StatusCreated, http.MethodPost, "/chat/channels", "bob", `{"name":"room","topic":"first"}`, &channel)
	deliver(t, h, bob1, bob2, carol1, carol2)
	if bob1.takeChannel(channel.ID) == 0 || bob2.takeChannel(channel.ID) == 0 {
		t.Fatal("creating the channel did not tell both of bob's sessions")
	}
	if carol1.takeChannel(channel.ID) != 0 || carol2.takeChannel(channel.ID) != 0 {
		t.Fatal("carol was told about a channel she is not in")
	}
	got, ok := listedChannel(t, h, "bob", channel.ID)
	if !ok || got.Name != "room" || got.Topic != "first" {
		t.Fatalf("bob lists %+v, want room", got)
	}
	if _, ok := listedChannel(t, h, "carol", channel.ID); ok {
		t.Fatal("carol lists a channel she was not added to")
	}

	path := "/chat/channels/" + strconv.FormatInt(channel.ID, 10)
	h.expect(t, http.StatusOK, http.MethodPatch, path, "bob", `{"name":"plans","topic":"sunday"}`, nil)
	deliver(t, h, bob1, bob2, carol1, carol2)
	if bob2.takeChannel(channel.ID) == 0 {
		t.Fatal("bob's other session was not told the channel was renamed")
	}
	got, ok = listedChannel(t, h, "bob", channel.ID)
	if !ok || got.Name != "plans" || got.Topic != "sunday" {
		t.Fatalf("after rename bob lists %+v", got)
	}

	h.expect(t, http.StatusOK, http.MethodPut, path+"/members", "bob", userBody(h.users["carol"], member...), nil)
	deliver(t, h, bob1, bob2, carol1, carol2)
	if carol1.takeChannel(channel.ID) == 0 || carol2.takeChannel(channel.ID) == 0 {
		t.Fatal("adding carol did not tell both of her sessions")
	}
	wantMembers := memberUserIDs(t, h, path, "bob")
	for _, as := range []string{"bob", "carol"} {
		if got := memberUserIDs(t, h, path, as); !slices.Equal(got, wantMembers) {
			t.Errorf("%s members %v, want %v", as, got, wantMembers)
		}
		if _, ok := listedChannel(t, h, as, channel.ID); !ok {
			t.Errorf("%s does not list the channel after the add", as)
		}
	}

	h.expect(t, http.StatusOK, http.MethodDelete, path+"/members", "carol", userBody(h.users["carol"]), nil)
	deliver(t, h, bob1, bob2, carol1, carol2)
	if carol1.takeChannel(channel.ID) == 0 || carol2.takeChannel(channel.ID) == 0 {
		t.Fatal("carol leaving did not tell both of her sessions")
	}
	if _, ok := listedChannel(t, h, "carol", channel.ID); ok {
		t.Fatal("carol still lists the channel after leaving")
	}
	h.expect(t, http.StatusNotFound, http.MethodGet, path+"/messages", "carol", "", nil)
	h.expect(t, http.StatusNotFound, http.MethodGet, path+"/members", "carol", "", nil)
	if _, ok := listedChannel(t, h, "bob", channel.ID); !ok {
		t.Fatal("bob lost the channel when carol left")
	}

	h.expect(t, http.StatusOK, http.MethodPut, path+"/members", "bob", userBody(h.users["carol"], member...), nil)
	deliver(t, h, bob1, bob2, carol1, carol2)
	if carol1.takeChannel(channel.ID) == 0 || carol2.takeChannel(channel.ID) == 0 {
		t.Fatal("re-adding carol did not tell both of her sessions")
	}
	// carol-2 is gone for the removal itself. Coming back, the channel list
	// is enough: she does not need the event she missed.
	carol2.disconnect()
	h.expect(t, http.StatusOK, http.MethodDelete, path+"/members", "bob", userBody(h.users["carol"]), nil)
	deliver(t, h, bob1, bob2, carol1)
	if carol1.takeChannel(channel.ID) == 0 {
		t.Fatal("removing carol did not tell the session that was connected")
	}
	carol2.reconnect(t)
	if _, ok := listedChannel(t, h, "carol", channel.ID); ok {
		t.Fatal("carol still lists the channel after being removed")
	}
	h.expect(t, http.StatusNotFound, http.MethodGet, path+"/messages", "carol", "", nil)
}

// TestChatConverge_ReadMarkerCatchUp is the catch-up half of a read marker.
// A session that misses chat_read_marker_changed can still read unreadCount
// off the channel list, but nothing returns lastReadMessageId. PUT
// /chat/channels/{id}/read moves the marker when the id is newer, so it is
// not a safe query. The body fails until one of those reads exists.
func TestChatConverge_ReadMarkerCatchUp(t *testing.T) {
	t.Skip("gap #3543: no read returns lastReadMessageId; GET /chat/channels has unreadCount only, and PUT /chat/channels/{id}/read moves the marker when given a newer id")

	h := newHarness(t)
	roomID, room := roomWithKey(t, h)
	messages := room + "/messages"
	h.expect(t, http.StatusOK, http.MethodPut, room+"/members", "bob", userBody(h.users["carol"], member...), nil)
	bob := openSession(t, h, "bob", "bob", false)
	carol1 := openSession(t, h, "carol", "carol-1", false)
	carol2 := openSession(t, h, "carol", "carol-2", false)
	posted := make([]chatutil.Message, 3)
	for i := range posted {
		posted[i] = postMsg(t, h, messages, "bob", byte('1'+i))
	}
	deliver(t, h, bob, carol1, carol2)
	carol2.disconnect()

	var marked chatutil.MarkReadResult
	h.expect(t, http.StatusOK, http.MethodPut, room+"/read", "carol", readBody(posted[1].ID), &marked)
	deliver(t, h, bob, carol1)
	if got := h.unreadCount(t, "bob", roomID); got != 0 {
		t.Fatalf("bob unread changed to %d while carol read", got)
	}

	carol2.reconnect(t)
	carol2.catchUp(t, messages)
	// Unread count survives on the channel list. The marker id does not,
	// unless a read of it was added.
	if got := h.unreadCount(t, "carol", roomID); got != marked.UnreadCount {
		t.Fatalf("carol unread on the list = %d, mark-read answered %d", got, marked.UnreadCount)
	}
	got, ok := readMarker(t, h, room, "carol", roomID)
	if !ok || got.LastReadMessageID != marked.LastReadMessageID || got.UnreadCount != marked.UnreadCount {
		t.Fatalf("restored marker %+v (ok %v), want %+v", got, ok, marked)
	}
	if carol2.lastRead != 0 && carol2.lastRead != marked.LastReadMessageID {
		t.Fatalf("carol-2 marker moved to %d without an event or a read", carol2.lastRead)
	}
}

// readMarker is a session's marker as a later read can see it: GET
// /chat/channels/{id}/read when that route exists, otherwise lastReadMessageId
// on the channel list. Neither exists today.
func readMarker(t *testing.T, h harness, room, as string, channelID int64) (chatutil.MarkReadResult, bool) {
	t.Helper()
	code, body := h.do(t, http.MethodGet, room+"/read", as, "")
	if code == http.StatusOK {
		var result chatutil.MarkReadResult
		if err := json.Unmarshal([]byte(body), &result); err != nil {
			t.Fatal(err)
		}
		return result, true
	}
	code, body = h.do(t, http.MethodGet, "/chat/channels", as, "")
	if code != http.StatusOK {
		t.Fatalf("list channels = %d %s", code, body)
	}
	var payload struct {
		Channels []map[string]any `json:"channels"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatal(err)
	}
	for _, channel := range payload.Channels {
		id, _ := channel["id"].(float64)
		if int64(id) != channelID {
			continue
		}
		raw, exists := channel["lastReadMessageId"]
		if !exists {
			return chatutil.MarkReadResult{}, false
		}
		n, isNum := raw.(float64)
		if !isNum {
			t.Fatalf("lastReadMessageId is %T", raw)
		}
		unread, _ := channel["unreadCount"].(float64)
		return chatutil.MarkReadResult{
			ChannelID: channelID, LastReadMessageID: int64(n), UnreadCount: int64(unread),
		}, true
	}
	t.Fatalf("channel %d missing from %s's list", channelID, as)
	return chatutil.MarkReadResult{}, false
}
