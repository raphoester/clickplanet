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
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
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

func (stubSubscriber) Stage() clicks.Stage { return clicks.WorldStage }

// earth1 names TestBoard.
type earth1 struct{}

func (earth1) Name(id clicks.BoardID) clicks.BoardName {
	if id != clicks.TestBoard {
		return ""
	}
	return "earth-1"
}

type silentNotices struct{}

func (silentNotices) Attend(charges.Holder, string) (<-chan notices.Event, func()) {
	return nil, func() {}
}

type addressedNotices map[charges.Holder]chan notices.Event

func (f addressedNotices) Attend(holder charges.Holder, _ string) (<-chan notices.Event, func()) {
	return f[holder], func() {}
}

// flagNotices hears the flag each stream attends with.
type flagNotices struct {
	news  chan notices.Event
	flags chan string
}

func (f flagNotices) Attend(_ charges.Holder, flag string) (<-chan notices.Event, func()) {
	f.flags <- flag
	return f.news, func() {}
}

// playsFor is the flag every caller plays for.
type playsFor string

func (p playsFor) FlagOf(context.Context, clicks.Payer) (string, error) { return string(p), nil }

type quietRules struct{}

func (quietRules) Watch() (<-chan cprules.Set[*planetv1.Rules], func()) {
	return nil, func() {}
}

type changingRules struct {
	changes chan cprules.Set[*planetv1.Rules]
}

func (c changingRules) Watch() (<-chan cprules.Set[*planetv1.Rules], func()) {
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

func TestAnUpdateIsCarriedToTheSink(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	updates <- clicks.Change{Version: clicks.MapVersion{Epoch: opening.Epoch, Seq: 42}, Update: &clicks.TileUpdate{Tile: 42, Value: "fr", Previous: "de"}}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, earth1{}, time.Hour, silentNotices{}, playsFor(""), quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{
		{Heartbeat: true, Version: opening, Board: "earth-1"},
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
			done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, earth1{}, time.Hour, silentNotices{}, playsFor(""), quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
		}()

		<-sink.fed
		<-sink.fed
		cancel()
		require.NoError(t, <-done)

		require.Equal(t, []listen_for_events_usecase.Event{
			{Heartbeat: true, Version: opening, Board: "earth-1"},
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
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, earth1{}, time.Hour, silentNotices{}, playsFor(""), quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening, Board: "earth-1"}, {Blast: blast}}, sink.seen())
}

func TestAFortificationIsCarriedToTheSinkAsOneFrame(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	fortification := &clicks.Fortification{Landmass: 1, Country: "fr", Tile: 7, Tiles: 3}
	updates <- clicks.Change{Fortification: fortification}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, earth1{}, time.Hour, silentNotices{}, playsFor(""), quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).
			Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening, Board: "earth-1"}, {Fortification: fortification}}, sink.seen())
}

func TestAGrantToTheCallersAccountIsCarriedToTheSinkAGuestsToo(t *testing.T) {
	news := make(chan notices.Event, 1)
	held := &charges.Gain{After: charges.Held{Bombs: 1, Shields: 20}}
	news <- notices.Event{Granted: held}

	guest := cpctx.AddAccountToContext(cpctx.AddIPToContext(t.Context(), "1.2.3.4"), "acc-1")
	ctx, cancel := context.WithCancel(guest)
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(
			stubSubscriber{updates: make(chan clicks.Change)}, earth1{}, time.Hour, addressedNotices{"acc-1": news}, playsFor(""), quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening, Board: "earth-1"}, {Granted: held}}, sink.seen())
}

func TestASpreadIsCarriedToTheSink(t *testing.T) {
	news := make(chan notices.Event, 1)
	spread := &notices.Spread{CountryID: "fr", Tile: 7, Neighbours: []uint32{6, 8}}
	news <- notices.Event{Spread: spread}

	ctx, cancel := context.WithCancel(cpctx.AddIPToContext(t.Context(), "1.2.3.4"))
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: make(chan clicks.Change)}, earth1{}, time.Hour,
			addressedNotices{charges.NoHolder: news}, playsFor(""), quietRules{},
			facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	for range 2 {
		<-sink.fed
	}
	cancel()
	require.NoError(t, <-done)

	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening, Board: "earth-1"}, {Spread: spread}}, sink.seen())
}

func TestATallyIsCarriedToTheSinkAtItsVersion(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	tally := &clicks.Tally{Countries: []clicks.FlagTiles{{Flag: "fr", Tiles: 3}}}
	updates <- clicks.Change{Version: opening, Tally: tally}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, earth1{}, time.Hour, silentNotices{}, playsFor(""), quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening, Board: "earth-1"}, {Version: opening, Tally: tally}}, sink.seen())
}

func TestAHeartbeatSaysTheVersionOfTheLastChangeTheStreamSent(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	sent := clicks.MapVersion{Epoch: opening.Epoch, Seq: 50}
	updates <- clicks.Change{Version: sent, Update: &clicks.TileUpdate{Tile: 1, Value: "fr"}}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: updates}, earth1{}, time.Millisecond, silentNotices{}, playsFor(""), quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
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

func (*watchedSubscriber) Stage() clicks.Stage { return clicks.WorldStage }

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

	require.NoError(t, listen_for_events_usecase.New(subscriber, earth1{}, time.Hour, silentNotices{}, playsFor(""), quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(t.Context(), sink))

	assert.Equal(t, []bool{true}, sink.followed, "a client resyncs on it, so nothing after it may be missed")
}

func TestASilentFeedKeepsSendingHeartbeats(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(
			stubSubscriber{updates: make(chan clicks.Change)}, earth1{}, time.Millisecond, silentNotices{}, playsFor(""), quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
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

	err := listen_for_events_usecase.New(stubSubscriber{updates: updates}, earth1{}, time.Hour, silentNotices{}, playsFor(""), quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).
		Execute(t.Context(), &recorder{})

	require.NoError(t, err, "the map going away is not the caller's error")
}

func TestAFailedSendEndsTheFeed(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	updates <- clicks.Change{Update: &clicks.TileUpdate{Tile: 1}}
	sink := &recorder{err: assert.AnError, accepted: 1}

	err := listen_for_events_usecase.New(stubSubscriber{updates: updates}, earth1{}, time.Hour, silentNotices{}, playsFor(""), quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).
		Execute(t.Context(), sink)

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, []listen_for_events_usecase.Event{
		{Heartbeat: true, Version: opening, Board: "earth-1"},
		{Update: &clicks.TileUpdate{Tile: 1}},
	}, sink.seen())
}

func TestAFailedOpeningHeartbeatEndsTheFeed(t *testing.T) {
	updates := make(chan clicks.Change, 1)
	updates <- clicks.Change{Update: &clicks.TileUpdate{Tile: 1}}
	sink := &recorder{err: assert.AnError}

	err := listen_for_events_usecase.New(stubSubscriber{updates: updates}, earth1{}, time.Hour, silentNotices{}, playsFor(""), quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).
		Execute(t.Context(), sink)

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening, Board: "earth-1"}}, sink.seen())
}

func TestAStreamIsToldWhenItOpensEvenOneThatFails(t *testing.T) {
	log := facts.NewLog(cptime.NewFixedClock(at))
	cause := errors.New("subscriber gone")

	err := listen_for_events_usecase.New(stubSubscriber{err: cause}, earth1{}, time.Hour, silentNotices{}, playsFor(""), quietRules{}, log).
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
	changes := make(chan cprules.Set[*planetv1.Rules], 1)
	changes <- cprules.Set[*planetv1.Rules]{}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{}, earth1{}, time.Hour, silentNotices{}, playsFor(""), changingRules{changes: changes},
			facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening, Board: "earth-1"}, {RulesChanged: true}}, sink.seen())
}

func TestAStreamHearsTheMinesOfTheFlagItsCallerPlaysFor(t *testing.T) {
	news := make(chan notices.Event, 1)
	laid := &notices.MineLaid{Flag: "fr", Tile: 9, Version: clicks.MapVersion{Epoch: 7, Seq: 2}}
	news <- notices.Event{MineLaid: laid}
	attended := flagNotices{news: news, flags: make(chan string, 1)}

	ctx, cancel := context.WithCancel(t.Context())
	sink := &recorder{fed: make(chan struct{}), stopped: ctx.Done()}
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{updates: make(chan clicks.Change)}, earth1{}, time.Hour, attended,
			playsFor("fr"), quietRules{}, facts.NewLog(cptime.NewFixedClock(at))).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	assert.Equal(t, "fr", <-attended.flags)
	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true, Version: opening, Board: "earth-1"}, {MineLaid: laid}}, sink.seen())
}
