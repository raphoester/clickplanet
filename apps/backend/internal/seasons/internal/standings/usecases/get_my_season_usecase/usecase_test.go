package get_my_season_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings/usecases/get_my_season_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	ada            = standings.AccountID{0: 1, 15: 1}
	bob            = standings.AccountID{0: 1, 15: 2}
)

func seasonZero() calendar.Calendar {
	return calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: seasonZeroEnds, Finale: 2 * time.Hour},
	}})
}

func take(t *testing.T, store *inmemory_contribution_store.Store, account standings.AccountID, country standings.Country, tiles int) {
	t.Helper()

	for range tiles {
		require.NoError(t, store.RecordTake(t.Context(), 0, standings.Take{Account: account, Country: country}))
	}
}

func TestThePlaceIsTheCallersInTheCurrentSeason(t *testing.T) {
	store := inmemory_contribution_store.New()
	players := standings.NewFakePlayers()
	players.Add(ada, standings.Player{Name: "Ada"})
	players.Add(bob, standings.Player{Name: "Bob"})
	take(t, store, ada, "fr", 2)
	take(t, store, bob, "de", 5)
	clock := cptime.NewFixedClock(seasonZeroEnds.Add(-time.Hour))
	useCase := get_my_season_usecase.New(seasonZero(), clock, standings.NewBoard(store, players))

	place, err := useCase.Execute(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, standings.Place{
		Line:       standings.Line{Account: ada, Country: "fr", Tiles: 2},
		GlobalRank: 2, CountryRank: 1,
	}, place)

	clock.Advance(time.Hour)
	place, err = useCase.Execute(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, standings.Place{}, place, "no season is no place")
}

func TestAFailureToReadThePlaceIsAnError(t *testing.T) {
	store := inmemory_contribution_store.New()
	refused := errors.New("the database is down")
	store.FailWith(refused)

	_, err := get_my_season_usecase.New(seasonZero(), cptime.NewFixedClock(seasonZeroEnds.Add(-time.Hour)),
		standings.NewBoard(store, standings.NewFakePlayers())).Execute(t.Context(), ada)

	assert.ErrorIs(t, err, refused)
}
