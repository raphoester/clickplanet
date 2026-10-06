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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
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

type quietGarrisons struct{}

func (quietGarrisons) Subscribe(context.Context) (<-chan garrisons.Garrison, error) {
	return make(chan garrisons.Garrison), nil
}

type stubGarrisons chan garrisons.Garrison

func (s stubGarrisons) Subscribe(context.Context) (<-chan garrisons.Garrison, error) {
	return s, nil
}

type recorder struct {
	mu     sync.Mutex
	events []listen_for_events_usecase.Event
	err    error
	fed    chan struct{}
}

func (r *recorder) Send(event listen_for_events_usecase.Event) error {
	r.mu.Lock()
	r.events = append(r.events, event)
	r.mu.Unlock()

	if r.fed != nil {
		r.fed <- struct{}{}
	}

	return r.err
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
		require.NoError(t, listen_for_events_usecase.New(stubSubscriber{}, time.Hour, feed, quietGarrisons{}).Execute(ctx, &recorder{}))
	}

	assert.Equal(t, []bonuses.Entrant{"account:a-player", "1.2.3.4"}, feed.entrants)
}

func TestAFailedSubscriptionEndsTheFeed(t *testing.T) {
	cause := errors.New("disk on fire")

	err := listen_for_events_usecase.New(stubSubscriber{err: cause}, time.Hour, silentFeed{}, quietGarrisons{}).
		Execute(t.Context(), &recorder{})

	require.ErrorIs(t, err, cause)
}

func TestAnUpdateIsCarriedToTheSink(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	updates <- clicks.Change{Update: &clicks.TileUpdate{Tile: 42, Value: "fr", Previous: "de"}}

	sink := &recorder{fed: make(chan struct{})}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentFeed{}, quietGarrisons{}).Execute(ctx, sink)
	}()

	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{
		{Update: clicks.TileUpdate{Tile: 42, Value: "fr", Previous: "de"}},
	}, sink.seen())
}

func TestAGarrisonIsCarriedToTheSink(t *testing.T) {
	defences := make(stubGarrisons, 1)
	defences <- garrisons.Garrison{Tile: 42, Country: "fr", Defenders: 3}

	sink := &recorder{fed: make(chan struct{})}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: make(chan clicks.Change)}, time.Hour, silentFeed{},
			defences).Execute(ctx, sink)
	}()

	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{
		{Garrison: &garrisons.Garrison{Tile: 42, Country: "fr", Defenders: 3}},
	}, sink.seen())
}

func TestABlastIsCarriedToTheSinkAsOneFrame(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	blast := &clicks.Blast{Tile: 7, CountryID: "fr", Cleared: []uint32{6, 7, 8}}
	updates <- clicks.Change{Blast: blast}

	sink := &recorder{fed: make(chan struct{})}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentFeed{}, quietGarrisons{}).Execute(ctx, sink)
	}()

	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Blast: blast}}, sink.seen())
}

func TestASilentFeedKeepsSendingHeartbeats(t *testing.T) {
	sink := &recorder{fed: make(chan struct{})}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(
			stubSubscriber{updates: make(chan clicks.Change)}, time.Millisecond, silentFeed{}, quietGarrisons{}).Execute(ctx, sink)
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

	err := listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentFeed{}, quietGarrisons{}).
		Execute(t.Context(), &recorder{})

	require.NoError(t, err, "the map going away is not the caller's error")
}

func TestAFailedSendEndsTheFeed(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	updates <- clicks.Change{Update: &clicks.TileUpdate{Tile: 1}}

	err := listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentFeed{}, quietGarrisons{}).
		Execute(t.Context(), &recorder{err: assert.AnError})

	require.ErrorIs(t, err, assert.AnError)
}
