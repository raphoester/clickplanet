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

type standingsService struct {
	scoringv1connect.UnimplementedSeasonServiceHandler
}

func (standingsService) GetStandings(
	context.Context,
	*connect.Request[scoringv1.GetStandingsRequest],
) (*connect.Response[scoringv1.GetStandingsResponse], error) {
	return connect.NewResponse(&scoringv1.GetStandingsResponse{}), nil
}

func (standingsService) GetMySeason(
	context.Context,
	*connect.Request[scoringv1.GetMySeasonRequest],
) (*connect.Response[scoringv1.GetMySeasonResponse], error) {
	return connect.NewResponse(&scoringv1.GetMySeasonResponse{}), nil
}

func TestTheStandingsAreAGetAnyCacheMayKeepForFifteenSecondsAndTheCallersSeasonIsNot(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(scoringv1connect.NewSeasonServiceHandler(standingsService{},
		connect.WithInterceptors(scoringv1controller.NewCacheInterceptor())))
	methods := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods <- r.Method
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	client := scoringv1connect.NewSeasonServiceClient(server.Client(), server.URL, connect.WithHTTPGet())

	standings, err := client.GetStandings(t.Context(), connect.NewRequest(&scoringv1.GetStandingsRequest{CountryId: "fr"}))
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, <-methods)
	assert.Equal(t, "public, max-age=15", standings.Header().Get("Cache-Control"))

	mine, err := client.GetMySeason(t.Context(), connect.NewRequest(&scoringv1.GetMySeasonRequest{}))
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, <-methods)
	assert.Empty(t, mine.Header().Get("Cache-Control"))
}
