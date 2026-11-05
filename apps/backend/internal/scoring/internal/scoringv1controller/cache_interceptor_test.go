package scoringv1controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller"
)

type service struct {
	scoringv1connect.UnimplementedScoringServiceHandler
}

func (service) GetBoard(context.Context, *connect.Request[scoringv1.GetBoardRequest]) (*connect.Response[scoringv1.GetBoardResponse], error) {
	return connect.NewResponse(&scoringv1.GetBoardResponse{}), nil
}

func (service) GetRound(context.Context, *connect.Request[scoringv1.GetRoundRequest]) (*connect.Response[scoringv1.GetRoundResponse], error) {
	return connect.NewResponse(&scoringv1.GetRoundResponse{}), nil
}

func (service) GetScores(context.Context, *connect.Request[scoringv1.GetScoresRequest]) (*connect.Response[scoringv1.GetScoresResponse], error) {
	return connect.NewResponse(&scoringv1.GetScoresResponse{}), nil
}

func (service) GetMyTally(context.Context, *connect.Request[scoringv1.GetMyTallyRequest]) (*connect.Response[scoringv1.GetMyTallyResponse], error) {
	return connect.NewResponse(&scoringv1.GetMyTallyResponse{}), nil
}

func TestTheReadsAreGetsAnyCacheMayKeepAndTheCallersTallyIsNot(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(scoringv1connect.NewScoringServiceHandler(service{},
		connect.WithInterceptors(scoringv1controller.NewCacheInterceptor())))
	methods := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods <- r.Method
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	client := scoringv1connect.NewScoringServiceClient(server.Client(), server.URL, connect.WithHTTPGet())

	board, err := client.GetBoard(t.Context(), connect.NewRequest(&scoringv1.GetBoardRequest{BookId: "season-0"}))
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, <-methods)
	assert.Equal(t, "public, max-age=15", board.Header().Get("Cache-Control"))

	round, err := client.GetRound(t.Context(), connect.NewRequest(&scoringv1.GetRoundRequest{RoundId: "season-0/day-24"}))
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, <-methods)
	assert.Equal(t, "public, max-age=10", round.Header().Get("Cache-Control"))

	scores, err := client.GetScores(t.Context(), connect.NewRequest(&scoringv1.GetScoresRequest{BookId: "season-0"}))
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, <-methods)
	assert.Equal(t, "public, max-age=10", scores.Header().Get("Cache-Control"))

	mine, err := client.GetMyTally(t.Context(), connect.NewRequest(&scoringv1.GetMyTallyRequest{BookId: "season-0"}))
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, <-methods)
	assert.Empty(t, mine.Header().Get("Cache-Control"))
}
