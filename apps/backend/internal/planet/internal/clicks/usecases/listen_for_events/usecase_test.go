package listen_for_events_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/listen_for_events"
)

type stubSubscriber struct {
	updates chan clicks.TileUpdate
	err     error
}

func (s stubSubscriber) Subscribe(context.Context) (<-chan clicks.TileUpdate, error) {
	return s.updates, s.err
}

// recorder stands in for the stream. Send is called from the use case's own
// goroutine while the test reads, so the events are guarded.
type recorder struct {
	mu     sync.Mutex
	events []listen_for_events.Event
	err    error
	fed    chan struct{}
}

func (r *recorder) Send(event listen_for_events.Event) error {
	r.mu.Lock()
	r.events = append(r.events, event)
	r.mu.Unlock()

	if r.fed != nil {
		r.fed <- struct{}{}
	}

	return r.err
}

func (r *recorder) seen() []listen_for_events.Event {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]listen_for_events.Event(nil), r.events...)
}

func TestAFailedSubscriptionEndsTheFeed(t *testing.T) {
	cause := errors.New("disk on fire")

	err := listen_for_events.New(stubSubscriber{err: cause}, time.Hour, nil).
		Execute(t.Context(), &recorder{})

	require.ErrorIs(t, err, cause)
}

func TestAnUpdateIsCarriedToTheSink(t *testing.T) {
	updates := make(chan clicks.TileUpdate, 1)
	updates <- clicks.TileUpdate{Tile: 42, Value: "fr", Previous: "de"}

	sink := &recorder{fed: make(chan struct{})}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events.New(stubSubscriber{updates: updates}, time.Hour, nil).Execute(ctx, sink)
	}()

	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events.Event{
		{Update: clicks.TileUpdate{Tile: 42, Value: "fr", Previous: "de"}},
	}, sink.seen())
}

// Cloudflare cuts a silent response at ~125s with a 524, so a quiet feed has to
// keep speaking or it dies and reconnects forever, losing each gap.
func TestASilentFeedKeepsSendingHeartbeats(t *testing.T) {
	sink := &recorder{fed: make(chan struct{})}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events.New(
			stubSubscriber{updates: make(chan clicks.TileUpdate)}, time.Millisecond, nil).Execute(ctx, sink)
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
	updates := make(chan clicks.TileUpdate)
	close(updates)

	err := listen_for_events.New(stubSubscriber{updates: updates}, time.Hour, nil).
		Execute(t.Context(), &recorder{})

	require.NoError(t, err, "the map going away is not the caller's error")
}

// A stream that has gone away ends the feed rather than being retried.
func TestAFailedSendEndsTheFeed(t *testing.T) {
	updates := make(chan clicks.TileUpdate, 1)
	updates <- clicks.TileUpdate{Tile: 1}

	err := listen_for_events.New(stubSubscriber{updates: updates}, time.Hour, nil).
		Execute(t.Context(), &recorder{err: assert.AnError})

	require.ErrorIs(t, err, assert.AnError)
}
