package listen_for_events_handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metagamev1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/metagame/v1/metagamev1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/metagamev1controller/listen_for_events_handler"
)

type stubTurns struct {
	turns chan *metagamev1.Turn

	mu  sync.Mutex
	ctx context.Context
}

func newStubTurns() *stubTurns {
	return &stubTurns{turns: make(chan *metagamev1.Turn, 4)}
}

func (s *stubTurns) Subscribe(ctx context.Context) <-chan *metagamev1.Turn {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ctx = ctx
	return s.turns
}

func (s *stubTurns) subscribed() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.ctx
}

func serve(t *testing.T, turns *stubTurns, heartbeat time.Duration) metagamev1connect.MetagameServiceClient {
	t.Helper()
	handler := listen_for_events_handler.New(turns, heartbeat)
	mux := http.NewServeMux()
	mux.Handle(metagamev1connect.MetagameServiceListenForEventsProcedure, connect.NewServerStreamHandler(
		metagamev1connect.MetagameServiceListenForEventsProcedure, handler.ListenForEvents))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return metagamev1connect.NewMetagameServiceClient(server.Client(), server.URL)
}

func TestTheStreamSendsEachTurnAsTheTurnCase(t *testing.T) {
	turns := newStubTurns()
	turns.turns <- &metagamev1.Turn{Kind: metagamev1.EntryKind_ENTRY_KIND_PLAIN}
	turns.turns <- &metagamev1.Turn{Kind: metagamev1.EntryKind_ENTRY_KIND_BATTLE, Description: "Triple points."}

	stream, err := serve(t, turns, time.Hour).ListenForEvents(t.Context(), connect.NewRequest(&metagamev1.ListenForEventsRequest{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })

	for _, want := range []metagamev1.EntryKind{metagamev1.EntryKind_ENTRY_KIND_PLAIN, metagamev1.EntryKind_ENTRY_KIND_BATTLE} {
		require.True(t, stream.Receive(), stream.Err())
		require.NotNil(t, stream.Msg().GetTurn(), "a turn travels as the turn case")
		assert.Equal(t, want, stream.Msg().GetTurn().GetKind())
	}
	assert.Equal(t, "Triple points.", stream.Msg().GetTurn().GetDescription())
}

func TestAQuietStreamSendsAHeartbeat(t *testing.T) {
	stream, err := serve(t, newStubTurns(), 10*time.Millisecond).ListenForEvents(t.Context(),
		connect.NewRequest(&metagamev1.ListenForEventsRequest{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })

	require.True(t, stream.Receive(), stream.Err())
	assert.NotNil(t, stream.Msg().GetHeartbeat())
}

func TestTheFeedIsLeftWhenTheStreamCloses(t *testing.T) {
	turns := newStubTurns()
	turns.turns <- &metagamev1.Turn{Kind: metagamev1.EntryKind_ENTRY_KIND_PLAIN}
	stream, err := serve(t, turns, time.Hour).ListenForEvents(t.Context(), connect.NewRequest(&metagamev1.ListenForEventsRequest{}))
	require.NoError(t, err)
	require.True(t, stream.Receive(), stream.Err())

	require.NoError(t, stream.Close())

	require.Eventually(t, func() bool { return turns.subscribed().Err() != nil }, 2*time.Second, time.Millisecond)
}

func TestAFeedThatClosesEndsTheStreamCleanly(t *testing.T) {
	turns := newStubTurns()
	turns.turns <- &metagamev1.Turn{Kind: metagamev1.EntryKind_ENTRY_KIND_PLAIN}
	stream, err := serve(t, turns, time.Hour).ListenForEvents(t.Context(), connect.NewRequest(&metagamev1.ListenForEventsRequest{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })
	require.True(t, stream.Receive(), stream.Err())

	close(turns.turns)

	assert.False(t, stream.Receive())
	require.NoError(t, stream.Err())
}
