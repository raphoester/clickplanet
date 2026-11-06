package listen_for_events_usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/announcements"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/reactions"
)

var (
	ada   = messages.AccountID{15: 1}
	bob   = messages.AccountID{15: 2}
	open  = messages.NewRooms(messages.FakeFlags{ada: "fr", bob: "de"})
	world = listen_for_events_usecase.In{Viewer: messages.NoAccount, Room: messages.World}
	team  = listen_for_events_usecase.In{Viewer: ada, Room: messages.FlagRoom("fr")}
)

type stubSubscriber struct {
	feed chan feed.Update
	err  error
}

func (s stubSubscriber) Subscribe(context.Context) (<-chan feed.Update, error) {
	return s.feed, s.err
}

type recorder struct {
	mu       sync.Mutex
	events   []listen_for_events_usecase.Event
	err      error
	accepted int
	fed      chan struct{}
}

func (r *recorder) Send(event listen_for_events_usecase.Event) error {
	r.mu.Lock()
	r.events = append(r.events, event)
	refused := len(r.events) > r.accepted
	r.mu.Unlock()

	if r.fed != nil {
		r.fed <- struct{}{}
	}

	if refused {
		return r.err
	}
	return nil
}

func (r *recorder) seen() []listen_for_events_usecase.Event {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]listen_for_events_usecase.Event(nil), r.events...)
}

func TestAFailedSubscriptionEndsTheFeed(t *testing.T) {
	cause := errors.New("disk on fire")

	err := listen_for_events_usecase.New(stubSubscriber{err: cause}, open, time.Hour).
		Execute(t.Context(), world, &recorder{})

	require.ErrorIs(t, err, cause)
}

func TestAMessageIsCarriedToTheSink(t *testing.T) {
	updates := make(chan feed.Update, 1)
	updates <- feed.MessageSent(messages.Restored(t, messages.Snapshot{ID: "message-1", Text: "hello"}))

	sink := &recorder{fed: make(chan struct{})}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{feed: updates}, open, time.Hour).Execute(ctx, world, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{
		{Heartbeat: true},
		{Update: feed.MessageSent(messages.Restored(t, messages.Snapshot{ID: "message-1", Text: "hello"}))},
	}, sink.seen())
}

func TestAStreamOpensWithAHeartbeat(t *testing.T) {
	sink := &recorder{fed: make(chan struct{})}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(
			stubSubscriber{feed: make(chan feed.Update)}, open, time.Hour).Execute(ctx, world, sink)
	}()

	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}}, sink.seen())
}

type watchedSubscriber struct{ subscribed bool }

func (s *watchedSubscriber) Subscribe(context.Context) (<-chan feed.Update, error) {
	s.subscribed = true

	updates := make(chan feed.Update)
	close(updates)

	return updates, nil
}

type witness struct {
	subscriber *watchedSubscriber
	followed   []bool
}

func (w *witness) Send(listen_for_events_usecase.Event) error {
	w.followed = append(w.followed, w.subscriber.subscribed)
	return nil
}

func TestTheOpeningHeartbeatComesOnceTheFeedIsFollowed(t *testing.T) {
	subscriber := &watchedSubscriber{}
	sink := &witness{subscriber: subscriber}

	require.NoError(t, listen_for_events_usecase.New(subscriber, open, time.Hour).Execute(t.Context(), world, sink))

	assert.Equal(t, []bool{true}, sink.followed, "a client resyncs on it, so nothing after it may be missed")
}

func TestASilentFeedKeepsSendingHeartbeats(t *testing.T) {
	sink := &recorder{fed: make(chan struct{})}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(
			stubSubscriber{feed: make(chan feed.Update)}, open, time.Millisecond).Execute(ctx, world, sink)
	}()

	for range 3 {
		<-sink.fed
	}
	cancel()
	require.NoError(t, <-done)

	for i, event := range sink.seen() {
		assert.Truef(t, event.Heartbeat, "frame %d is not a heartbeat", i)
	}
}

func TestTheFeedEndsWhenTheSubscriptionCloses(t *testing.T) {
	updates := make(chan feed.Update)
	close(updates)

	err := listen_for_events_usecase.New(stubSubscriber{feed: updates}, open, time.Hour).
		Execute(t.Context(), world, &recorder{})

	require.NoError(t, err)
}

func TestAFailedSendEndsTheFeed(t *testing.T) {
	sent := feed.MessageSent(messages.Restored(t, messages.Snapshot{ID: "message-1"}))
	updates := make(chan feed.Update, 1)
	updates <- sent
	sink := &recorder{err: assert.AnError, accepted: 1}

	err := listen_for_events_usecase.New(stubSubscriber{feed: updates}, open, time.Hour).
		Execute(t.Context(), world, sink)

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}, {Update: sent}}, sink.seen())
}

func TestAFailedOpeningHeartbeatEndsTheFeed(t *testing.T) {
	updates := make(chan feed.Update, 1)
	updates <- feed.MessageSent(messages.Restored(t, messages.Snapshot{ID: "message-1"}))
	sink := &recorder{err: assert.AnError}

	err := listen_for_events_usecase.New(stubSubscriber{feed: updates}, open, time.Hour).
		Execute(t.Context(), world, sink)

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}}, sink.seen())
}

func said(t *testing.T, id messages.MessageID, room messages.Room) feed.Update {
	t.Helper()
	return feed.MessageSent(messages.Restored(t, messages.Snapshot{ID: id, Room: room, Text: "hi"}))
}

func listened(t *testing.T, in listen_for_events_usecase.In, sent ...feed.Update) ([]listen_for_events_usecase.Event, error) {
	t.Helper()

	updates := make(chan feed.Update, len(sent))
	for _, update := range sent {
		updates <- update
	}
	close(updates)
	sink := &recorder{accepted: len(sent) + 1}

	err := listen_for_events_usecase.New(stubSubscriber{feed: updates}, open, time.Hour).Execute(t.Context(), in, sink)
	return sink.seen(), err
}

func TestAFlagsStreamCarriesWhatIsSaidInItsRoomAlone(t *testing.T) {
	plan := said(t, "plan", messages.FlagRoom("fr"))
	gone := feed.MessageRemoved("anything")

	events, err := listened(t, team,
		said(t, "hello", messages.World),
		plan,
		said(t, "plan-de", messages.FlagRoom("de")),
		feed.ReactionsChanged(messages.Restored(t, messages.Snapshot{ID: "hello", Room: messages.World}), reactions.Tally{}),
		gone,
	)

	require.NoError(t, err)
	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}, {Update: plan}, {Update: gone}}, events)
}

func TestTheChatEveryoneReadsCarriesNothingSaidInAFlagsRoom(t *testing.T) {
	hello := said(t, "hello", messages.World)

	events, err := listened(t, world, said(t, "plan", messages.FlagRoom("fr")), hello)

	require.NoError(t, err)
	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}, {Update: hello}}, events)
}

func announced(id byte, room messages.Room) feed.Update {
	return feed.Announced(announcements.NewAnnouncement(announcements.AnnouncementID{15: id}, announcements.KindFlagMission,
		time.Time{}, json.RawMessage(`{"country":"fr"}`)).WithRoom(room))
}

func TestAFlagsStreamCarriesTheAnnouncementsToldInItsRoomAlone(t *testing.T) {
	ours := announced(1, messages.FlagRoom("fr"))

	events, err := listened(t, team, announced(2, messages.World), ours, announced(3, messages.FlagRoom("de")))

	require.NoError(t, err)
	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}, {Update: ours}}, events)
}

func TestTheChatEveryoneReadsCarriesEveryAnnouncement(t *testing.T) {
	everyone, ours, theirs := announced(1, messages.World), announced(2, messages.FlagRoom("fr")), announced(3, messages.FlagRoom("de"))

	events, err := listened(t, world, everyone, ours, theirs)

	require.NoError(t, err)
	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}, {Update: everyone}, {Update: ours}, {Update: theirs}}, events)
}

func TestAStreamForTheRoomOfAnotherFlagIsRefusedBeforeAnythingIsSent(t *testing.T) {
	events, err := listened(t, listen_for_events_usecase.In{Viewer: bob, Room: messages.FlagRoom("fr")})

	require.ErrorIs(t, err, messages.ErrNotInRoom)
	assert.Empty(t, events)
}

func TestAFlagsStreamEndsOnceItsViewerPledgesAnotherFlag(t *testing.T) {
	events, err := listened(t, team,
		feed.RoomLeft(bob, messages.FlagRoom("fr")),
		feed.RoomLeft(ada, messages.FlagRoom("de")),
		feed.RoomLeft(ada, messages.FlagRoom("fr")),
		said(t, "plan", messages.FlagRoom("fr")),
	)

	require.ErrorIs(t, err, messages.ErrNotInRoom)
	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}}, events, "nothing said after it")
}
