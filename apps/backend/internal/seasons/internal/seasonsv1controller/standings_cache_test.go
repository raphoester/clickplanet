package seasonsv1controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller"
)

type standingsService struct {
	seasonsv1connect.UnimplementedSeasonServiceHandler
}

func (standingsService) GetStandings(
	context.Context,
	*connect.Request[seasonsv1.GetStandingsRequest],
) (*connect.Response[seasonsv1.GetStandingsResponse], error) {
	return connect.NewResponse(&seasonsv1.GetStandingsResponse{}), nil
}

func (standingsService) GetMySeason(
	context.Context,
	*connect.Request[seasonsv1.GetMySeasonRequest],
) (*connect.Response[seasonsv1.GetMySeasonResponse], error) {
	return connect.NewResponse(&seasonsv1.GetMySeasonResponse{}), nil
}

func TestTheStandingsAreAGetAnyCacheMayKeepForFifteenSecondsAndTheCallersSeasonIsNot(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(seasonsv1connect.NewSeasonServiceHandler(standingsService{},
		connect.WithInterceptors(seasonsv1controller.NewCacheInterceptor())))
	methods := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods <- r.Method
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	client := seasonsv1connect.NewSeasonServiceClient(server.Client(), server.URL, connect.WithHTTPGet())

	standings, err := client.GetStandings(t.Context(), connect.NewRequest(&seasonsv1.GetStandingsRequest{CountryId: "fr"}))
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, <-methods)
	assert.Equal(t, "public, max-age=15", standings.Header().Get("Cache-Control"))

	mine, err := client.GetMySeason(t.Context(), connect.NewRequest(&seasonsv1.GetMySeasonRequest{}))
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, <-methods)
	assert.Empty(t, mine.Header().Get("Cache-Control"))
}
