package listen_for_events_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/feed/usecases/listen_for_events_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
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

	err := listen_for_events_usecase.New(stubSubscriber{err: cause}, time.Hour).
		Execute(t.Context(), &recorder{})

	require.ErrorIs(t, err, cause)
}

func TestAMessageIsCarriedToTheSink(t *testing.T) {
	updates := make(chan feed.Update, 1)
	updates <- feed.MessageSent(messages.NewMessage("message-1", time.Time{}, messages.NoAccount, "", "hello"))

	sink := &recorder{fed: make(chan struct{})}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(stubSubscriber{feed: updates}, time.Hour).Execute(ctx, sink)
	}()

	<-sink.fed
	<-sink.fed
	cancel()
	require.NoError(t, <-done)

	require.Equal(t, []listen_for_events_usecase.Event{
		{Heartbeat: true},
		{Update: feed.MessageSent(messages.NewMessage("message-1", time.Time{}, messages.NoAccount, "", "hello"))},
	}, sink.seen())
}

func TestAStreamOpensWithAHeartbeat(t *testing.T) {
	sink := &recorder{fed: make(chan struct{})}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(
			stubSubscriber{feed: make(chan feed.Update)}, time.Hour).Execute(ctx, sink)
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

	require.NoError(t, listen_for_events_usecase.New(subscriber, time.Hour).Execute(t.Context(), sink))

	assert.Equal(t, []bool{true}, sink.followed, "a client resyncs on it, so nothing after it may be missed")
}

func TestASilentFeedKeepsSendingHeartbeats(t *testing.T) {
	sink := &recorder{fed: make(chan struct{})}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- listen_for_events_usecase.New(
			stubSubscriber{feed: make(chan feed.Update)}, time.Millisecond).Execute(ctx, sink)
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

	err := listen_for_events_usecase.New(stubSubscriber{feed: updates}, time.Hour).
		Execute(t.Context(), &recorder{})

	require.NoError(t, err)
}

func TestAFailedSendEndsTheFeed(t *testing.T) {
	sent := feed.MessageSent(messages.NewMessage("message-1", time.Time{}, messages.NoAccount, "", ""))
	updates := make(chan feed.Update, 1)
	updates <- sent
	sink := &recorder{err: assert.AnError, accepted: 1}

	err := listen_for_events_usecase.New(stubSubscriber{feed: updates}, time.Hour).
		Execute(t.Context(), sink)

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}, {Update: sent}}, sink.seen())
}

func TestAFailedOpeningHeartbeatEndsTheFeed(t *testing.T) {
	updates := make(chan feed.Update, 1)
	updates <- feed.MessageSent(messages.NewMessage("message-1", time.Time{}, messages.NoAccount, "", ""))
	sink := &recorder{err: assert.AnError}

	err := listen_for_events_usecase.New(stubSubscriber{feed: updates}, time.Hour).
		Execute(t.Context(), sink)

	require.ErrorIs(t, err, assert.AnError)
	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}}, sink.seen())
}
