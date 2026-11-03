package listen_for_events_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inprocess_payment_feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/get_missions_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/listen_for_events_usecase"
)

var ada = missions.AccountID{15: 1}

type countingDays struct {
	mu    sync.Mutex
	reads []missions.AccountID
	err   error
}

func (d *countingDays) Execute(_ context.Context, in get_missions_usecase.In) (get_missions_usecase.Out, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.reads = append(d.reads, in.Account)
	return get_missions_usecase.Out{}, d.err
}

type recordingSink struct {
	mu         sync.Mutex
	days       int
	heartbeats int
	err        error
}

func (s *recordingSink) SendDay(get_missions_usecase.Out) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.days++
	return s.err
}

func (s *recordingSink) SendHeartbeat() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.heartbeats++
	return s.err
}

func (s *recordingSink) sent() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.days, s.heartbeats
}

func listen(
	ctx context.Context, feed *inprocess_payment_feed.Feed, days *countingDays, sink *recordingSink, heartbeat time.Duration,
) <-chan error {
	done := make(chan error, 1)
	go func() { done <- listen_for_events_usecase.New(feed, days, heartbeat).Execute(ctx, ada, sink) }()
	return done
}

func TestTheDayIsSentAtOnceThenAgainAtEachPayment(t *testing.T) {
	feed := inprocess_payment_feed.New()
	days, sink := &countingDays{}, &recordingSink{}
	ctx, cancel := context.WithCancel(t.Context())
	done := listen(ctx, feed, days, sink, time.Hour)

	require.Eventually(t, func() bool { sent, _ := sink.sent(); return sent == 1 }, time.Second, time.Millisecond)
	feed.Publish(ada)
	require.Eventually(t, func() bool { sent, _ := sink.sent(); return sent == 2 }, time.Second, time.Millisecond)
	cancel()

	require.NoError(t, <-done)
	assert.Equal(t, []missions.AccountID{ada, ada}, days.reads)
}

func TestAnotherAccountsPaymentSendsNothing(t *testing.T) {
	feed := inprocess_payment_feed.New()
	days, sink := &countingDays{}, &recordingSink{}
	ctx, cancel := context.WithCancel(t.Context())
	done := listen(ctx, feed, days, sink, time.Hour)

	require.Eventually(t, func() bool { sent, _ := sink.sent(); return sent == 1 }, time.Second, time.Millisecond)
	feed.Publish(missions.AccountID{15: 2})
	time.Sleep(20 * time.Millisecond)
	cancel()

	require.NoError(t, <-done)
	sent, _ := sink.sent()
	assert.Equal(t, 1, sent)
}

func TestAQuietStreamGetsHeartbeats(t *testing.T) {
	feed := inprocess_payment_feed.New()
	sink := &recordingSink{}
	ctx, cancel := context.WithCancel(t.Context())
	done := listen(ctx, feed, &countingDays{}, sink, time.Millisecond)

	require.Eventually(t, func() bool { _, beats := sink.sent(); return beats >= 2 }, time.Second, time.Millisecond)
	cancel()

	require.NoError(t, <-done)
}

func TestAFailedReadEndsTheStream(t *testing.T) {
	failed := errors.New("postgres is gone")

	err := <-listen(t.Context(), inprocess_payment_feed.New(), &countingDays{err: failed}, &recordingSink{}, time.Hour)

	require.ErrorIs(t, err, failed)
}

func TestASinkFailureEndsTheStream(t *testing.T) {
	gone := errors.New("the stream is gone")

	err := <-listen(t.Context(), inprocess_payment_feed.New(), &countingDays{}, &recordingSink{err: gone}, time.Hour)

	require.ErrorIs(t, err, gone)
}
