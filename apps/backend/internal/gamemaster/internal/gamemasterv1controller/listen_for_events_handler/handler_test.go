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

	gamemasterv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/gamemaster/v1/gamemasterv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/gamemasterv1controller/listen_for_events_handler"
)

type stubTurns struct {
	turns chan *gamemasterv1.GetTurnResponse

	mu  sync.Mutex
	ctx context.Context
}

func newStubTurns() *stubTurns {
	return &stubTurns{turns: make(chan *gamemasterv1.GetTurnResponse, 4)}
}

func (s *stubTurns) Subscribe(ctx context.Context) <-chan *gamemasterv1.GetTurnResponse {
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

func serve(t *testing.T, turns *stubTurns, heartbeat time.Duration) gamemasterv1connect.GamemasterServiceClient {
	t.Helper()
	handler := listen_for_events_handler.New(turns, heartbeat)
	mux := http.NewServeMux()
	mux.Handle(gamemasterv1connect.GamemasterServiceListenForEventsProcedure, connect.NewServerStreamHandler(
		gamemasterv1connect.GamemasterServiceListenForEventsProcedure, handler.ListenForEvents))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return gamemasterv1connect.NewGamemasterServiceClient(server.Client(), server.URL)
}

func now(kind gamemasterv1.EntryKind, description string, battles ...string) *gamemasterv1.GetTurnResponse {
	timeline := &gamemasterv1.Timeline{}
	for _, id := range battles {
		timeline.Battles = append(timeline.Battles, &gamemasterv1.Battle{EntryId: id})
	}
	return &gamemasterv1.GetTurnResponse{Turn: &gamemasterv1.Turn{Kind: kind, Description: description}, Timeline: timeline}
}

func open(t *testing.T, turns *stubTurns) *connect.ServerStreamForClient[gamemasterv1.GamemasterEvent] {
	t.Helper()
	stream, err := serve(t, turns, time.Hour).ListenForEvents(t.Context(), connect.NewRequest(&gamemasterv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })
	return stream
}

func receive(t *testing.T, stream *connect.ServerStreamForClient[gamemasterv1.GamemasterEvent]) *gamemasterv1.GamemasterEvent {
	t.Helper()
	require.True(t, stream.Receive(), stream.Err())
	return stream.Msg()
}

const (
	plain  = gamemasterv1.EntryKind_ENTRY_KIND_PLAIN
	battle = gamemasterv1.EntryKind_ENTRY_KIND_BATTLE
)

func TestTheStreamSendsTheTurnThenTheTimelineFirst(t *testing.T) {
	turns := newStubTurns()
	turns.turns <- now(plain, "", "finale")
	stream := open(t, turns)

	assert.Equal(t, plain, receive(t, stream).GetTurn().GetKind(), "a turn travels as the turn case")
	assert.Equal(t, "finale", receive(t, stream).GetTimeline().GetBattles()[0].GetEntryId(), "the timeline as its own case")
}

func TestTheStreamSendsOnlyWhatChanged(t *testing.T) {
	turns := newStubTurns()
	turns.turns <- now(plain, "", "finale")
	turns.turns <- now(plain, "", "finale", "rush")
	turns.turns <- now(battle, "Triple points.", "finale", "rush")
	stream := open(t, turns)
	receive(t, stream)
	receive(t, stream)

	assert.Len(t, receive(t, stream).GetTimeline().GetBattles(), 2, "a battle added sends the timeline alone")
	assert.Equal(t, "Triple points.", receive(t, stream).GetTurn().GetDescription(), "a battle that starts sends the turn alone")
}

func TestAQuietStreamSendsAHeartbeat(t *testing.T) {
	stream, err := serve(t, newStubTurns(), 10*time.Millisecond).ListenForEvents(t.Context(),
		connect.NewRequest(&gamemasterv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })

	require.True(t, stream.Receive(), stream.Err())
	assert.NotNil(t, stream.Msg().GetHeartbeat())
}

func TestTheFeedIsLeftWhenTheStreamCloses(t *testing.T) {
	turns := newStubTurns()
	turns.turns <- now(plain, "")
	stream, err := serve(t, turns, time.Hour).ListenForEvents(t.Context(), connect.NewRequest(&gamemasterv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	require.True(t, stream.Receive(), stream.Err())

	require.NoError(t, stream.Close())

	require.Eventually(t, func() bool { return turns.subscribed().Err() != nil }, 2*time.Second, time.Millisecond)
}

func TestAFeedThatClosesEndsTheStreamCleanly(t *testing.T) {
	turns := newStubTurns()
	turns.turns <- now(plain, "")
	stream := open(t, turns)
	receive(t, stream)
	receive(t, stream)

	close(turns.turns)

	assert.False(t, stream.Receive())
	require.NoError(t, stream.Err())
}
