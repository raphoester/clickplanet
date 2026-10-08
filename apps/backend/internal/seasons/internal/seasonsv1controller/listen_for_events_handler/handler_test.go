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

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
)

type stubBoards struct {
	boards chan *seasonsv1.Board

	mu    sync.Mutex
	asked []string
	ctx   context.Context
}

func newStubBoards() *stubBoards {
	return &stubBoards{boards: make(chan *seasonsv1.Board, 4)}
}

func (s *stubBoards) Subscribe(ctx context.Context, country string) <-chan *seasonsv1.Board {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.asked = append(s.asked, country)
	s.ctx = ctx
	return s.boards
}

func (s *stubBoards) subscribed() ([]string, context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.asked, s.ctx
}

type stubRaces struct {
	races chan *seasonsv1.Race

	mu       sync.Mutex
	followed int
}

func newStubRaces() *stubRaces {
	return &stubRaces{races: make(chan *seasonsv1.Race, 4)}
}

func (s *stubRaces) Subscribe(context.Context) <-chan *seasonsv1.Race {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.followed++
	return s.races
}

func (s *stubRaces) followers() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.followed
}

func serve(t *testing.T, boards *stubBoards, races *stubRaces, heartbeat time.Duration) seasonsv1connect.SeasonServiceClient {
	t.Helper()
	handler := listen_for_events_handler.New(boards, races, cpcountries.New(), heartbeat)
	mux := http.NewServeMux()
	mux.Handle(seasonsv1connect.SeasonServiceListenForEventsProcedure, connect.NewServerStreamHandler(
		seasonsv1connect.SeasonServiceListenForEventsProcedure, handler.ListenForEvents))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return seasonsv1connect.NewSeasonServiceClient(server.Client(), server.URL)
}

func board(names ...string) *seasonsv1.Board {
	shown := &seasonsv1.Board{}
	for _, name := range names {
		shown.Standings = append(shown.Standings, &seasonsv1.Standing{Name: name})
	}
	return shown
}

func TestTheStreamSendsEachBoardOfTheViewItAsksFor(t *testing.T) {
	boards := newStubBoards()
	boards.boards <- board("Ana")
	boards.boards <- board("Ana", "Kofi")

	stream, err := serve(t, boards, newStubRaces(), time.Hour).ListenForEvents(t.Context(),
		connect.NewRequest(&seasonsv1.ListenForEventsRequest{CountryId: "fr"}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })

	for _, want := range []int{1, 2} {
		require.True(t, stream.Receive(), stream.Err())
		require.NotNil(t, stream.Msg().GetBoard(), "a board travels as the board case")
		assert.Len(t, stream.Msg().GetBoard().GetStandings(), want)
	}
	asked, _ := boards.subscribed()
	assert.Equal(t, []string{"fr"}, asked)
}

func TestAQuietStreamSendsAHeartbeat(t *testing.T) {
	stream, err := serve(t, newStubBoards(), newStubRaces(), 10*time.Millisecond).ListenForEvents(t.Context(),
		connect.NewRequest(&seasonsv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })

	require.True(t, stream.Receive(), stream.Err())
	assert.NotNil(t, stream.Msg().GetHeartbeat())
}

func TestEveryStreamSendsEachRaceWhateverItsView(t *testing.T) {
	races := newStubRaces()
	races.races <- &seasonsv1.Race{Scores: []*seasonsv1.Score{{Rank: 1, CountryId: "fr", Points: 25}}}

	stream, err := serve(t, newStubBoards(), races, time.Hour).ListenForEvents(t.Context(),
		connect.NewRequest(&seasonsv1.ListenForEventsRequest{CountryId: "de"}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })

	require.True(t, stream.Receive(), stream.Err())
	require.NotNil(t, stream.Msg().GetRace(), "a race travels as the race case")
	assert.Equal(t, "fr", stream.Msg().GetRace().GetScores()[0].GetCountryId())
}

func TestACountryThatIsNotOneIsInvalidArgumentAndFollowsNothing(t *testing.T) {
	boards := newStubBoards()
	races := newStubRaces()
	stream, err := serve(t, boards, races, time.Hour).ListenForEvents(t.Context(),
		connect.NewRequest(&seasonsv1.ListenForEventsRequest{CountryId: "zz"}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })

	assert.False(t, stream.Receive())
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(stream.Err()))
	asked, _ := boards.subscribed()
	assert.Empty(t, asked)
	assert.Zero(t, races.followers())
}

func TestTheViewIsLeftWhenTheStreamCloses(t *testing.T) {
	boards := newStubBoards()
	boards.boards <- board("Ana")
	stream, err := serve(t, boards, newStubRaces(), time.Hour).ListenForEvents(t.Context(),
		connect.NewRequest(&seasonsv1.ListenForEventsRequest{}))
	require.NoError(t, err)
	require.True(t, stream.Receive(), stream.Err())

	require.NoError(t, stream.Close())

	_, followed := boards.subscribed()
	require.Eventually(t, func() bool { return followed.Err() != nil }, 2*time.Second, time.Millisecond)
}
