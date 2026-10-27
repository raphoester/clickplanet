package scoringv1controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1/scoringv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar/usecases/get_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_race_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/scoringv1controller/get_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestTheSeasonIsAGetAnyCacheMayKeepForAMinute(t *testing.T) {
	ends := time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	service := scoringv1controller.SeasonService{
		GetSeasonHandler: get_season_handler.New(get_season_usecase.New(
			calendar.New(calendar.Config{}, calendar.Window{Number: 0, StartsAt: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), EndsAt: ends}),
			cptime.NewFixedClock(ends.Add(-time.Hour)),
		)),
	}
	mux := http.NewServeMux()
	mux.Handle(scoringv1connect.NewSeasonServiceHandler(service,
		connect.WithInterceptors(scoringv1controller.NewCacheInterceptor())))
	methods := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods <- r.Method
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	client := scoringv1connect.NewSeasonServiceClient(server.Client(), server.URL, connect.WithHTTPGet())
	res, err := client.GetSeason(t.Context(), connect.NewRequest(&scoringv1.GetSeasonRequest{}))
	require.NoError(t, err)

	assert.Equal(t, http.MethodGet, <-methods)
	assert.Equal(t, "public, max-age=60", res.Header().Get("Cache-Control"))
	assert.Equal(t, ends.UnixMilli(), res.Msg.GetSeason().GetEndsAtUnixMs())
}

type race struct{}

func (race) Race(context.Context) (*scoringv1.Race, error) {
	return &scoringv1.Race{Closed: &scoringv1.ClosedRound{Season: 1, Number: 5}}, nil
}

func TestTheRaceIsAGetAnyCacheMayKeepForTenSeconds(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(scoringv1connect.NewSeasonServiceHandler(scoringv1controller.SeasonService{GetRaceHandler: get_race_handler.New(race{})},
		connect.WithInterceptors(scoringv1controller.NewCacheInterceptor())))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := scoringv1connect.NewSeasonServiceClient(server.Client(), server.URL, connect.WithHTTPGet())
	res, err := client.GetRace(t.Context(), connect.NewRequest(&scoringv1.GetRaceRequest{}))
	require.NoError(t, err)

	assert.Equal(t, "public, max-age=10", res.Header().Get("Cache-Control"))
	assert.Equal(t, uint32(5), res.Msg.GetRace().GetClosed().GetNumber())
}
