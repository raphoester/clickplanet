package seasonsv1controller_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1/seasonsv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar/usecases/get_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_hall_of_fame_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_race_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestTheSeasonIsAGetAnyCacheMayKeepForAMinute(t *testing.T) {
	ends := time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	service := seasonsv1controller.SeasonService{
		GetSeasonHandler: get_season_handler.New(get_season_usecase.New(
			calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: ends}}}),
			cptime.NewFixedClock(ends.Add(-time.Hour)),
		)),
	}
	mux := http.NewServeMux()
	mux.Handle(seasonsv1connect.NewSeasonServiceHandler(service,
		connect.WithInterceptors(seasonsv1controller.NewCacheInterceptor())))
	methods := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods <- r.Method
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	client := seasonsv1connect.NewSeasonServiceClient(server.Client(), server.URL, connect.WithHTTPGet())
	res, err := client.GetSeason(t.Context(), connect.NewRequest(&seasonsv1.GetSeasonRequest{}))
	require.NoError(t, err)

	assert.Equal(t, http.MethodGet, <-methods)
	assert.Equal(t, "public, max-age=60", res.Header().Get("Cache-Control"))
	assert.Equal(t, ends.UnixMilli(), res.Msg.GetSeason().GetEndsAtUnixMs())
}

type race struct{}

func (race) Race(context.Context) (*seasonsv1.Race, error) {
	return &seasonsv1.Race{Closed: &seasonsv1.ClosedRound{Season: 1, Number: 5}}, nil
}

func TestTheRaceIsAGetAnyCacheMayKeepForTenSeconds(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(seasonsv1connect.NewSeasonServiceHandler(seasonsv1controller.SeasonService{GetRaceHandler: get_race_handler.New(race{})},
		connect.WithInterceptors(seasonsv1controller.NewCacheInterceptor())))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := seasonsv1connect.NewSeasonServiceClient(server.Client(), server.URL, connect.WithHTTPGet())
	res, err := client.GetRace(t.Context(), connect.NewRequest(&seasonsv1.GetRaceRequest{}))
	require.NoError(t, err)

	assert.Equal(t, "public, max-age=10", res.Header().Get("Cache-Control"))
	assert.Equal(t, uint32(5), res.Msg.GetRace().GetClosed().GetNumber())
}

type hall struct{}

func (hall) HallOfFame(context.Context) (*seasonsv1.GetHallOfFameResponse, error) {
	return &seasonsv1.GetHallOfFameResponse{Seasons: []*seasonsv1.EndedSeason{{Number: 0}}}, nil
}

func TestTheHallOfFameIsAGetAnyCacheMayKeepForAMinute(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(seasonsv1connect.NewSeasonServiceHandler(seasonsv1controller.SeasonService{GetHallOfFameHandler: get_hall_of_fame_handler.New(hall{})},
		connect.WithInterceptors(seasonsv1controller.NewCacheInterceptor())))
	methods := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods <- r.Method
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)

	client := seasonsv1connect.NewSeasonServiceClient(server.Client(), server.URL, connect.WithHTTPGet())
	res, err := client.GetHallOfFame(t.Context(), connect.NewRequest(&seasonsv1.GetHallOfFameRequest{}))
	require.NoError(t, err)

	assert.Equal(t, http.MethodGet, <-methods)
	assert.Equal(t, "public, max-age=60", res.Header().Get("Cache-Control"))
	assert.Len(t, res.Msg.GetSeasons(), 1)
}
