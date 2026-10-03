package get_standings_handler_test

import (
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_standings_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/get_standings_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	ada            = standings.AccountID{0: 1, 15: 1}
)

func handler(t *testing.T, store *inmemory_contribution_store.Store) get_standings_handler.GetStandingsHandler {
	t.Helper()

	players := standings.NewFakePlayers()
	players.Add(ada, standings.Player{Name: "Ada", Color: standings.Color(playerv1.NameColor_NAME_COLOR_PINK)})
	return get_standings_handler.New(get_standings_usecase.New(
		calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: seasonZeroEnds, Finale: 2 * time.Hour}}}),
		cptime.NewFixedClock(seasonZeroEnds.Add(-time.Hour)), cpcountries.New(), standings.NewBoard(store, players),
	))
}

func TestEachStandingCarriesItsRankNameColorFlagAndTiles(t *testing.T) {
	store := inmemory_contribution_store.New()
	for range 3 {
		require.NoError(t, store.RecordTake(t.Context(), 0, standings.Take{Account: ada, Country: "fr"}))
	}

	res, err := handler(t, store).GetStandings(t.Context(), connect.NewRequest(&seasonsv1.GetStandingsRequest{CountryId: "fr"}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(&seasonsv1.GetStandingsResponse{Standings: []*seasonsv1.Standing{{
		Rank: 1, Name: "Ada", Color: playerv1.NameColor_NAME_COLOR_PINK, CountryId: "fr", Tiles: 3,
	}}}, res.Msg), res.Msg)
}

func TestACountryThatIsNotOneIsInvalidArgument(t *testing.T) {
	_, err := handler(t, inmemory_contribution_store.New()).GetStandings(t.Context(),
		connect.NewRequest(&seasonsv1.GetStandingsRequest{CountryId: "zz"}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestAFailureIsLeftToTheErrorNet(t *testing.T) {
	store := inmemory_contribution_store.New()
	refused := errors.New("the database is down")
	store.FailWith(refused)

	_, err := handler(t, store).GetStandings(t.Context(), connect.NewRequest(&seasonsv1.GetStandingsRequest{}))

	require.ErrorIs(t, err, refused)
	assert.Equal(t, connect.CodeUnknown, connect.CodeOf(err))
}
