package listen_for_events_usecase_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed/usecases/listen_for_events_usecase"
)

var rennes = catalog.Snapshot{
	ID: "r-0000000001", Name: "Rennes", State: catalog.Open, DecidedAt: time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC),
}

type sink struct {
	mu     sync.Mutex
	events []listen_for_events_usecase.Event
}

func (s *sink) Send(event listen_for_events_usecase.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *sink) sent() []listen_for_events_usecase.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]listen_for_events_usecase.Event(nil), s.events...)
}

func listening(t *testing.T, board *feed.Board, heartbeat time.Duration) (*sink, func() error) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	out := &sink{}
	done := make(chan error, 1)
	go func() { done <- listen_for_events_usecase.New(board, heartbeat).Execute(ctx, out) }()
	require.Eventually(t, func() bool { return len(out.sent()) > 0 }, time.Second, time.Millisecond)
	return out, func() error {
		cancel()
		return <-done
	}
}

func TestAStreamSaysItIsLiveThenSendsEachFlagTold(t *testing.T) {
	board := feed.NewBoard(8)
	out, stop := listening(t, board, time.Hour)

	board.Publish(feed.ToldAsItIs(rennes))
	require.Eventually(t, func() bool { return len(out.sent()) == 2 }, time.Second, time.Millisecond)
	require.NoError(t, stop())

	sent := out.sent()
	assert.Equal(t, listen_for_events_usecase.Event{Heartbeat: true}, sent[0])
	require.NotNil(t, sent[1].Flag)
	assert.Equal(t, rennes, *sent[1].Flag)
}

func TestAStreamSendsAFlagGone(t *testing.T) {
	board := feed.NewBoard(8)
	out, stop := listening(t, board, time.Hour)

	board.Publish(feed.ToldGone("r-0000000001"))
	require.Eventually(t, func() bool { return len(out.sent()) == 2 }, time.Second, time.Millisecond)
	require.NoError(t, stop())

	require.NotNil(t, out.sent()[1].Gone)
	assert.Equal(t, catalog.FlagID("r-0000000001"), *out.sent()[1].Gone)
	assert.Nil(t, out.sent()[1].Flag)
}

func TestAQuietStreamGetsHeartbeats(t *testing.T) {
	out, stop := listening(t, feed.NewBoard(8), 10*time.Millisecond)

	require.Eventually(t, func() bool { return len(out.sent()) >= 3 }, time.Second, time.Millisecond)
	require.NoError(t, stop())
	for _, event := range out.sent() {
		assert.True(t, event.Heartbeat)
	}
}

// The board closes a stream that falls behind: its page opens it again and reads the list.
type closing struct{}

func (closing) Attend() (<-chan feed.Told, func()) {
	flags := make(chan feed.Told)
	close(flags)
	return flags, func() {}
}

func TestAStreamTheBoardClosedEnds(t *testing.T) {
	out := &sink{}

	require.NoError(t, listen_for_events_usecase.New(closing{}, time.Hour).Execute(t.Context(), out))

	assert.Equal(t, []listen_for_events_usecase.Event{{Heartbeat: true}}, out.sent())
}
