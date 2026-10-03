package get_my_season_handler_test

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/caller"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/seasonsv1controller/get_my_season_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/get_my_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const adaID = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)

func handler(t *testing.T, named bool) get_my_season_handler.GetMySeasonHandler {
	t.Helper()

	ada, err := standings.AccountIDOf(adaID)
	require.NoError(t, err)
	bob := standings.AccountID{0: 1, 15: 2}
	store := inmemory_contribution_store.New()
	players := standings.NewFakePlayers()
	players.Add(ada, standings.Player{Name: "Ada", Guest: !named})
	players.Add(bob, standings.Player{Name: "Bob"})
	for _, take := range []standings.Take{
		{Account: ada, Country: "fr"},
		{Account: ada, Country: "fr"},
		{Account: ada, Country: "de"},
		{Account: bob, Country: "de"},
		{Account: bob, Country: "de"},
		{Account: bob, Country: "de"},
	} {
		require.NoError(t, store.RecordTake(t.Context(), 0, take))
	}

	return get_my_season_handler.New(get_my_season_usecase.New(
		calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: seasonZeroEnds, Finale: 2 * time.Hour}}}),
		cptime.NewFixedClock(seasonZeroEnds.Add(-time.Hour)), standings.NewBoard(store, players),
	))
}

func TestTheCallerReadsItsMainFlagItsTilesAndItsRanks(t *testing.T) {
	res, err := handler(t, true).GetMySeason(cpctx.AddAccountToContext(t.Context(), adaID),
		connect.NewRequest(&seasonsv1.GetMySeasonRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(&seasonsv1.GetMySeasonResponse{CountryId: "fr", Tiles: 2, GlobalRank: 2, CountryRank: 1}, res.Msg),
		res.Msg)
}

func TestAGuestReadsItsTilesAndNoRank(t *testing.T) {
	res, err := handler(t, false).GetMySeason(cpctx.AddAccountToContext(t.Context(), adaID),
		connect.NewRequest(&seasonsv1.GetMySeasonRequest{}))

	require.NoError(t, err)
	assert.True(t, proto.Equal(&seasonsv1.GetMySeasonResponse{CountryId: "fr", Tiles: 2}, res.Msg), res.Msg)
}

func TestACallWithNoAccountIsUnauthenticated(t *testing.T) {
	_, err := handler(t, true).GetMySeason(t.Context(), connect.NewRequest(&seasonsv1.GetMySeasonRequest{}))

	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	assert.ErrorIs(t, err, caller.ErrNoAccount)
}
