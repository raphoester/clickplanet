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

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_board_handler/board_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/listen_for_events_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
)

type stubFeed[K comparable, V any] struct {
	mu     sync.Mutex
	values map[K]chan V
	asked  []K
	ctx    context.Context
}

func newStubFeed[K comparable, V any]() *stubFeed[K, V] {
	return &stubFeed[K, V]{values: map[K]chan V{}}
}

func (s *stubFeed[K, V]) of(key K) chan V {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.values[key] == nil {
		s.values[key] = make(chan V, 4)
	}
	return s.values[key]
}

func (s *stubFeed[K, V]) Subscribe(ctx context.Context, key K) <-chan V {
	values := s.of(key)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.asked = append(s.asked, key)
	s.ctx = ctx
	return values
}

func (s *stubFeed[K, V]) subscribed() ([]K, context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.asked, s.ctx
}

type feeds struct {
	boards *stubFeed[board_query.View, *scoringv1.Board]
	scores *stubFeed[books.BookID, *scoringv1.Scores]
	rounds *stubFeed[books.RoundID, *scoringv1.Round]
}

func newFeeds() feeds {
	return feeds{
		boards: newStubFeed[board_query.View, *scoringv1.Board](),
		scores: newStubFeed[books.BookID, *scoringv1.Scores](),
		rounds: newStubFeed[books.RoundID, *scoringv1.Round](),
	}
}

func serve(t *testing.T, feeds feeds, heartbeat time.Duration) scoringv1connect.ScoringServiceClient {
	t.Helper()
	handler := listen_for_events_handler.New(listen_for_events_handler.Feeds{
		Boards: feeds.boards, Scores: feeds.scores, Rounds: feeds.rounds,
	}, cpcountries.New(), heartbeat)
	mux := http.NewServeMux()
	mux.Handle(scoringv1connect.ScoringServiceListenForEventsProcedure, connect.NewServerStreamHandler(
		scoringv1connect.ScoringServiceListenForEventsProcedure, handler.ListenForEvents,
	))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return scoringv1connect.NewScoringServiceClient(server.Client(), server.URL)
}

func listen(t *testing.T, feeds feeds, heartbeat time.Duration, req *scoringv1.ListenForEventsRequest) *connect.ServerStreamForClient[scoringv1.ScoringEvent] {
	t.Helper()
	stream, err := serve(t, feeds, heartbeat).ListenForEvents(t.Context(), connect.NewRequest(req))
	require.NoError(t, err)
	t.Cleanup(func() { _ = stream.Close() })
	return stream
}

func board(names ...string) *scoringv1.Board {
	shown := &scoringv1.Board{BookId: "season-0"}
	for _, name := range names {
		shown.Standings = append(shown.Standings, &scoringv1.Standing{Name: name})
	}
	return shown
}

func TestTheStreamSendsEachBoardOfTheViewItAsksFor(t *testing.T) {
	feeds := newFeeds()
	view := board_query.View{Book: "season-0", Country: "fr"}
	feeds.boards.of(view) <- board("Ana")
	feeds.boards.of(view) <- board("Ana", "Kofi")

	stream := listen(t, feeds, time.Hour, &scoringv1.ListenForEventsRequest{BookId: "season-0", CountryId: "fr"})

	for _, want := range []int{1, 2} {
		require.True(t, stream.Receive(), stream.Err())
		require.NotNil(t, stream.Msg().GetBoard(), "a board travels as the board case")
		assert.Len(t, stream.Msg().GetBoard().GetStandings(), want)
	}
	asked, _ := feeds.boards.subscribed()
	assert.Equal(t, []board_query.View{view}, asked)
	scores, _ := feeds.scores.subscribed()
	assert.Equal(t, []books.BookID{"season-0"}, scores)
}

func TestTheStreamSendsTheBooksScoresAndEachRoundAsked(t *testing.T) {
	feeds := newFeeds()
	feeds.scores.of("season-0") <- &scoringv1.Scores{BookId: "season-0"}
	feeds.rounds.of("season-0/day-24") <- &scoringv1.Round{RoundId: "season-0/day-24"}
	feeds.rounds.of("battle") <- &scoringv1.Round{RoundId: "battle"}

	stream := listen(t, feeds, time.Hour, &scoringv1.ListenForEventsRequest{
		BookId: "season-0", RoundIds: []string{"season-0/day-24", "battle", "battle"},
	})

	var got []string
	for range 3 {
		require.True(t, stream.Receive(), stream.Err())
		switch event := stream.Msg().GetEvent().(type) {
		case *scoringv1.ScoringEvent_Scores:
			got = append(got, "scores "+event.Scores.GetBookId())
		case *scoringv1.ScoringEvent_Round:
			got = append(got, "round "+event.Round.GetRoundId())
		default:
			require.Fail(t, "unexpected event", "%v", stream.Msg())
		}
	}
	assert.ElementsMatch(t, []string{"scores season-0", "round season-0/day-24", "round battle"}, got)
	rounds, _ := feeds.rounds.subscribed()
	assert.Equal(t, []books.RoundID{"season-0/day-24", "battle"}, rounds, "a round asked twice is followed once")
}

func TestAStreamWithNoBookFollowsNoBoardAndNoScores(t *testing.T) {
	feeds := newFeeds()
	stream := listen(t, feeds, 10*time.Millisecond, &scoringv1.ListenForEventsRequest{})

	require.True(t, stream.Receive(), stream.Err())
	assert.NotNil(t, stream.Msg().GetHeartbeat(), "a quiet stream sends a heartbeat")
	asked, _ := feeds.boards.subscribed()
	assert.Empty(t, asked)
}

func TestARequestThatIsNotOneIsInvalidArgumentAndFollowsNothing(t *testing.T) {
	many := make([]string, listen_for_events_handler.MaxRounds+1)
	for i := range many {
		many[i] = string(rune('a' + i))
	}
	for _, req := range []*scoringv1.ListenForEventsRequest{
		{BookId: "season-0", CountryId: "zz"},
		{BookId: string(make([]byte, 65))},
		{BookId: "season-0", RoundIds: []string{""}},
		{BookId: "season-0", RoundIds: many},
	} {
		feeds := newFeeds()
		stream := listen(t, feeds, time.Hour, req)

		assert.False(t, stream.Receive())
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(stream.Err()), "%v", req)
		asked, _ := feeds.boards.subscribed()
		assert.Empty(t, asked)
	}
}

func TestTheViewIsLeftWhenTheStreamCloses(t *testing.T) {
	feeds := newFeeds()
	feeds.boards.of(board_query.View{Book: "season-0"}) <- board("Ana")
	stream, err := serve(t, feeds, time.Hour).ListenForEvents(t.Context(),
		connect.NewRequest(&scoringv1.ListenForEventsRequest{BookId: "season-0"}))
	require.NoError(t, err)
	require.True(t, stream.Receive(), stream.Err())

	require.NoError(t, stream.Close())

	_, followed := feeds.boards.subscribed()
	require.Eventually(t, func() bool { return followed.Err() != nil }, 2*time.Second, time.Millisecond)
}
