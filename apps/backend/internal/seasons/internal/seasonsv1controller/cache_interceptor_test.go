package seasonsv1controller_test

import (
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
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func TestTheSeasonIsAGetAnyCacheMayKeepForAMinute(t *testing.T) {
	ends := time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	service := seasonsv1controller.SeasonService{
		GetSeasonHandler: get_season_handler.New(get_season_usecase.New(
			calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: ends, Finale: 2 * time.Hour}}}),
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
