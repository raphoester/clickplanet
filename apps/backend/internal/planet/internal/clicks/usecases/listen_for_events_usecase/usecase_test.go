package listen_for_events_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/notices"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 10, 26, 20, 0, 0, 0, time.UTC)

var opening = clicks.MapVersion{Epoch: 1_790_000_000, Seq: 41}

type stubSubscriber struct {
	updates chan clicks.Change
	err     error
}

func (s stubSubscriber) Subscribe(context.Context) (<-chan clicks.Change, clicks.MapVersion, error) {
	return s.updates, opening, s.err
}

type silentNotices struct{}

func (silentNotices) Attend(clicks.Caller) (<-chan notices.Event, func()) {
	return nil, func() {}
}

type addressedNotices map[clicks.Caller]chan notices.Event

func (f addressedNotices) Attend(caller clicks.Caller) (<-chan notices.Event, func()) {
	return f[caller], func() {}
}

type quietRules struct{}

func (quietRules) Watch() (<-chan rules.Set, func()) {
	return nil, func() {}
}

type changingRules struct{ changes chan rules.Set }

func (c changingRules) Watch() (<-chan rules.Set, func()) {
	return c.changes, func() {}
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

func TestAFailedSubscriptionEndsTheFeed(t *testing.T) {
	cause := errors.New("disk on fire")

	err := listen_for_events_usecase.New(stubSubscriber{err: cause}, time.Hour, silentNotices{}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).
		Execute(t.Context(), &recorder{})

	require.ErrorIs(t, err, cause)
}

func TestAnUpdateIsCarriedToTheSink(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	updates <- clicks.Change{Version: clicks.MapVersion{Epoch: opening.Epoch, Seq: 42}, Update: &clicks.TileUpdate{Tile: 42, Value: "fr", Previous: "de"}}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentNotices{}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{
		{Heartbeat: true, Version: opening},
		{Version: clicks.MapVersion{Epoch: opening.Epoch, Seq: 42}, Update: &clicks.TileUpdate{Tile: 42, Value: "fr", Previous: "de"}},
	}, sink.seen())
}

func TestAnEnclosureIsCarriedToTheSinkAsOneFrameAndOnlyItsCloserIsToldItIsTheirs(t *testing.T) {
	address := cpctx.AddIPToContext(t.Context(), "1.2.3.4")
	closer := cpctx.AddLinkedToContext(cpctx.AddAccountToContext(address, "a-player"))
	other := cpctx.AddLinkedToContext(cpctx.AddAccountToContext(address, "another-player"))
	version := clicks.MapVersion{Epoch: opening.Epoch, Seq: 42}
	enclosure := &clicks.Enclosure{Country: "fr", ClosingTile: 9, Filled: []uint32{4, 5}, By: clicks.PayerOf(closer)}

	for _, stream := range []struct {
		ctx   context.Context
		yours bool
	}{{closer, true}, {other, false}} {
		updates := make(chan clicks.Change, 1)
		updates <- clicks.Change{Version: version, Enclosure: enclosure}

		ctx, cancel := context.WithCancel(stream.ctx)
		sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
		done := make(chan error, 1)
		go func() {
			done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentNotices{}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
		}()

		<-sink.fed
		<-sink.fed
		cancel()
		require.NoError(t, <-done)

		require.Equal(t, []listen_for_events_usecase.Event{
			{Heartbeat: true, Version: opening},
			{Version: version, Enclosure: enclosure, Yours: stream.yours},
		}, sink.seen())
	}
}

func TestABlastIsCarriedToTheSinkAsOneFrame(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	blast := &clicks.Blast{Tile: 7, CountryID: "fr", Cleared: []uint32{6, 7, 8}}
	updates <- clicks.Change{Blast: blast}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentNotices{}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening}, {Blast: blast}}, sink.seen())
}

func TestAFortificationIsCarriedToTheSinkAsOneFrame(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	fortification := &clicks.Fortification{Landmass: 1, Country: "fr", Tile: 7, Tiles: 3}
	updates <- clicks.Change{Fortification: fortification}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentNotices{}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).
			Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening}, {Fortification: fortification}}, sink.seen())
}

func TestAGrantToTheSignedInCallerIsCarriedToTheSink(t *testing.T) {
	news := make(chan notices.Event, 1)
	held := &charges.Held{Bomb: true, Shields: 20}
	news <- notices.Event{Granted: held}

	signedIn := cpctx.AddLinkedToContext(cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "acc-1"))
	ctx, cancel := context.WithCancel(signedIn)
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(
			stubSubscriber{updates: make(chan clicks.Change)}, time.Hour, addressedNotices{"account:acc-1": news}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening}, {Granted: held}}, sink.seen())
}

func TestASpreadIsCarriedToTheSink(t *testing.T) {
	news := make(chan notices.Event, 1)
	spread := &notices.Spread{CountryID: "fr", Tile: 7, Neighbours: []uint32{6, 8}}
	news <- notices.Event{Spread: spread}

	ctx, cancel := context.WithCancel(cpctx.AddIPToContext(t.Context(), "1.2.3.4"))
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: make(chan clicks.Change)}, time.Hour,
			addressedNotices{"1.2.3.4": news}, quietRules{},
			facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	for range 2 {
		<-sink.fed
	}
	cancel()
	require.NoError(t, <-done)

	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening}, {Spread: spread}}, sink.seen())
}

func TestAStreamOpensWithAHeartbeat(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(
			stubSubscriber{updates: make(chan clicks.Change)}, time.Hour, silentNotices{}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening}}, sink.seen())
}

func TestATallyIsCarriedToTheSinkAtItsVersion(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	tally := &clicks.Tally{Countries: []clicks.FlagTiles{{Flag: "fr", Tiles: 3}}}
	updates <- clicks.Change{Version: opening, Tally: tally}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentNotices{}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening}, {Version: opening, Tally: tally}}, sink.seen())
}

func TestAHeartbeatSaysTheVersionOfTheLastChangeTheStreamSent(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	sent := clicks.MapVersion{Epoch: opening.Epoch, Seq: 50}
	updates <- clicks.Change{Version: sent, Update: &clicks.TileUpdate{Tile: 1, Value: "fr"}}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Millisecond, silentNotices{}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	for range 3 {
		<-sink.fed
	}
	cancel()
	require.NoError(t, <-done)

	seen := sink.seen()
	assert.Equal(t, opening, seen[0].Version, "the opening one says where the subscription started")
	assert.Equal(t, sent, seen[1].Version)
	assert.Equal(t, listen_for_events_usecase.Event{Heartbeat: true, Version: sent}, seen[2])
}

type watchedSubscriber struct{ subscribed bool }

func (s *watchedSubscriber) Subscribe(context.Context) (<-chan clicks.Change, clicks.MapVersion, error) {
	s.subscribed = true

	updates := make(chan clicks.Change)
	close(updates)

	return updates, opening, nil
}

type witness struct {
	subscriber *watchedSubscriber
	followed   []bool
}

func (w *witness) Send(listen_for_events_usecase.Event) error {
	w.followed = append(w.followed, w.subscriber.subscribed)
	return nil
}

func TestTheOpeningHeartbeatComesOnceTheMapIsFollowed(t *testing.T) {
	subscriber := &watchedSubscriber{}
	sink := &witness{subscriber: subscriber}

	require.NoError(t, listen_for_events_usecase.New(subscriber, time.Hour, silentNotices{}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(t.Context(), sink))

	assert.Equal(t, []bool{true}, sink.followed, "a client resyncs on it, so nothing after it may be missed")
}

func TestASilentFeedKeepsSendingHeartbeats(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(
			stubSubscriber{updates: make(chan clicks.Change)}, time.Millisecond, silentNotices{}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
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

	err := listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentNotices{}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).
		Execute(t.Context(), &recorder{})

	require.NoError(t, err, "the map going away is not the caller's error")
}

func TestAFailedSendEndsTheFeed(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	updates <- clicks.Change{Update: &clicks.TileUpdate{Tile: 1}}
	sink := &recorder{err: assert.AnError, accepted: 1}

	err := listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentNotices{}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).
		Execute(t.Context(), sink)

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, []listen_for_events_usecase.Event{
		{Heartbeat: true, Version: opening},
		{Update: &clicks.TileUpdate{Tile: 1}},
	}, sink.seen())
}

func TestAFailedOpeningHeartbeatEndsTheFeed(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	updates <- clicks.Change{Update: &clicks.TileUpdate{Tile: 1}}
	sink := &recorder{err: assert.AnError}

	err := listen_for_events_usecase.New(stubSubscriber{updates: updates}, time.Hour, silentNotices{}, quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).
		Execute(t.Context(), sink)

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening}}, sink.seen())
}

func TestAStreamIsToldWhenItOpensEvenOneThatFails(t *testing.T) {
	log := facts.NewLog(cptime.NewFixedClock(at))
	cause := errors.New("subscriber gone")

	err := listen_for_events_usecase.New(stubSubscriber{err: cause}, time.Hour, silentNotices{}, quietRules{}, log).
		Execute(cpctx.AddIPToContext(t.Context(), "2001:db8::9"), &recorder{})

	require.ErrorIs(t, err, cause)
	told := log.Take()
	require.Len(t, told, 1)
	opened, ok := told[0].(*planetv1.StreamOpened)
	require.True(t, ok)
	assert.Equal(t, "2001:db8::/64", opened.GetScope())
	assert.Equal(t, at, opened.GetOpenedAt().AsTime())
}

func TestAChangeOfRulesIsCarriedToTheSink(t *testing.T) {
	changes := make(chan rules.Set, 1)
	changes <- rules.Set{}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{}, time.Hour, silentNotices{}, changingRules{changes: changes},
			facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening}, {RulesChanged: true}}, sink.seen())
}
