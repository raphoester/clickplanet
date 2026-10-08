package listen_for_events_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
)

type stubSubscriber struct {
	updates chan clicks.Change
	err     error
}

func (s stubSubscriber) Subscribe(context.Context) (<-chan clicks.Change, error) {
	return s.updates, s.err
}

type silentFeed struct{}

func (silentFeed) Attend(bonuses.Entrant) (<-chan bonuses.Event, func()) {
	return nil, func() {}
}

type recorder struct {
	mu       sync.Mutex
	events   []listen_for_events_usecase.Event
	err      error
	accepted int
	fed      chan struct{}
	stopped  <-chan struct{}
}

func (r *recorder) Send(event listen_for_events_usecase.Event) error {
	r.mu.Lock()
	r.events = append(r.events, event)
	refused := len(r.events) > r.accepted
	r.mu.Unlock()

	if r.fed != nil {
		select {
		case r.fed <- struct{}{}:
		case <-r.stopped:
		}
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

type attendedFeed struct{ entrants []bonuses.Entrant }

func (f *attendedFeed) Attend(entrant bonuses.Entrant) (<-chan bonuses.Event, func()) {
	f.entrants = append(f.entrants, entrant)

	closed := make(chan bonuses.Event)
	close(closed)

	return closed, func() {}
}

func TestAStreamListensForTheBoxesOfTheEntrantThatOpenedIt(t *testing.T) {
	address := cpctx.AddIPToContext(t.Context(), "1.2.3.4")
	linked := cpctx.AddLinkedToContext(cpctx.AddAccountToContext(address, "a-player"))
	guest := cpctx.AddAccountToContext(address, "a-guest")
	feed := &attendedFeed{}

	for _, ctx := range []context.Context{linked, guest} {
		require.NoError(t, listen_for_events_usecase.New(stubSubscriber{}, time.Hour, feed).Execute(ctx, &recorder{}))
	}

	assert.Equal(t, []bonuses.Entrant{"account:a-player", "1.2.3.4"}, feed.entrants)
}

func TestAFailedSubscriptionEndsTheFeed(t *testing.T) {
	cause := errors.New("disk on fire")

	err := listen_for_events_usecase.New(stubSubscriber{err: cause}, time.Hour, silentFeed{}).
		Execute(t.Context(), &recorder{})

	require.ErrorIs(t, err, cause)
}

func TestAnUpdateIsCarriedToTheSink(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	updates <- clicks.Change{Update: &clicks.TileUpdate{Tile: 42, Value: "fr", Previous: "de"}}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentFeed{}).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{
		{Heartbeat: true},
		{Update: clicks.TileUpdate{Tile: 42, Value: "fr", Previous: "de"}},
	}, sink.seen())
}

func TestABlastIsCarriedToTheSinkAsOneFrame(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	blast := &clicks.Blast{Tile: 7, CountryID: "fr", Cleared: []uint32{6, 7, 8}}
	updates <- clicks.Change{Blast: blast}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentFeed{}).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}, {Blast: blast}}, sink.seen())
}

func TestAStreamOpensWithAHeartbeat(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(
			stubSubscriber{updates: make(chan clicks.Change)}, time.Hour, silentFeed{}).Execute(ctx, sink)
	}()

	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}}, sink.seen())
}

type watchedSubscriber struct{ subscribed bool }

func (s *watchedSubscriber) Subscribe(context.Context) (<-chan clicks.Change, error) {
	s.subscribed = true

	updates := make(chan clicks.Change)
	close(updates)

	return updates, nil
}

type witness struct {
	subscriber *watchedSubscriber
	feed       *attendedFeed
	followed   []bool
}

func (w *witness) Send(listen_for_events_usecase.Event) error {
	w.followed = append(w.followed, w.subscriber.subscribed && len(w.feed.entrants) == 1)
	return nil
}

func TestTheOpeningHeartbeatComesOnceTheMapAndTheBoxesAreFollowed(t *testing.T) {
	subscriber := &watchedSubscriber{}
	feed := &attendedFeed{}
	sink := &witness{subscriber: subscriber, feed: feed}

	require.NoError(t, listen_for_events_usecase.New(subscriber, time.Hour, feed).Execute(t.Context(), sink))

	assert.Equal(t, []bool{true}, sink.followed, "a client resyncs on it, so nothing after it may be missed")
}

func TestASilentFeedKeepsSendingHeartbeats(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(
			stubSubscriber{updates: make(chan clicks.Change)}, time.Millisecond, silentFeed{}).Execute(ctx, sink)
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
	updates := make(chan clicks.Change)
	close(updates)

	err := listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentFeed{}).
		Execute(t.Context(), &recorder{})

	require.NoError(t, err, "the map going away is not the caller's error")
}

func TestAFailedSendEndsTheFeed(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	updates <- clicks.Change{Update: &clicks.TileUpdate{Tile: 1}}
	sink := &recorder{err: assert.AnError, accepted: 1}

	err := listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentFeed{}).
		Execute(t.Context(), sink)

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, []listen_for_events_usecase.Event{
		{Heartbeat: true},
		{Update: clicks.TileUpdate{Tile: 1}},
	}, sink.seen())
}

func TestAFailedOpeningHeartbeatEndsTheFeed(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	updates <- clicks.Change{Update: &clicks.TileUpdate{Tile: 1}}
	sink := &recorder{err: assert.AnError}

	err := listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentFeed{}).
		Execute(t.Context(), sink)

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}}, sink.seen())
}
