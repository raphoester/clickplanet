package finales_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/finales"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

var (
	ada = players.AccountID{15: 1}
	bob = players.AccountID{15: 2}
)

func TestEachFighterFightsUnderItsFlagsPlaceAndAFlagOffTheTableHasNone(t *testing.T) {
	fights, err := finales.Result{
		Season:   0,
		Table:    []finales.Place{{Country: "fr", Rank: 1}, {Country: "de", Rank: 1}, {Country: "it", Rank: 3}},
		Fighters: []finales.Fighter{{Account: ada, Country: "de", Tiles: 5}, {Account: bob, Country: "tv", Tiles: 9}},
	}.Fights()

	require.NoError(t, err)
	require.Len(t, fights, 2)
	assert.Equal(t, finales.FightSnapshot{Account: ada, Season: 0, Country: "de", Rank: 1, TilesRank: 2},
		fights[0].TakeSnapshot(), "a tie shares the rank")
	assert.Equal(t, finales.FightSnapshot{Account: bob, Season: 0, Country: "tv", Rank: 0, TilesRank: 1}, fights[1].TakeSnapshot())
}

func TestEachFighterIsRankedByTheTilesItTookAndATieSharesTheRank(t *testing.T) {
	carol := players.AccountID{15: 3}
	dan := players.AccountID{15: 4}

	fights, err := finales.Result{
		Fighters: []finales.Fighter{
			{Account: ada, Country: "fr", Tiles: 40},
			{Account: bob, Country: "de", Tiles: 90},
			{Account: carol, Country: "fr", Tiles: 40},
			{Account: dan, Country: "it", Tiles: 7},
		},
	}.Fights()

	require.NoError(t, err)
	ranks := map[players.AccountID]uint32{}
	for _, fight := range fights {
		ranks[fight.Account()] = fight.TakeSnapshot().TilesRank
	}
	assert.Equal(t, map[players.AccountID]uint32{bob: 1, ada: 2, carol: 2, dan: 4}, ranks)
}

func TestAFighterWithNoFlagIsRefused(t *testing.T) {
	_, err := finales.Result{Fighters: []finales.Fighter{{Account: ada}}}.Fights()

	assert.ErrorIs(t, err, finales.ErrNoCountry)
}

func TestFightsSayWhichFinaleWasFoughtAndWhereItsFlagPlaced(t *testing.T) {
	fight, err := finales.FightSnapshot{Account: ada, Season: 0, Country: "fr", Rank: 2}.Restore()
	require.NoError(t, err)
	fights := finales.Fights{fight}

	assert.True(t, fights.FoughtIn(0))
	assert.False(t, fights.FoughtIn(1))
	assert.True(t, fights.PlacedIn(0, 2))
	assert.False(t, fights.PlacedIn(0, 1))
	assert.False(t, fights.PlacedIn(1, 2))
	assert.False(t, finales.Fights{}.FoughtIn(0))
}

func TestFightsSayWhereTheFighterRankedByTiles(t *testing.T) {
	fight, err := finales.FightSnapshot{Account: ada, Season: 0, Country: "fr", Rank: 2, TilesRank: 3}.Restore()
	require.NoError(t, err)
	fights := finales.Fights{fight}

	assert.True(t, fights.RankedByTilesIn(0, 3))
	assert.False(t, fights.RankedByTilesIn(0, 2))
	assert.False(t, fights.RankedByTilesIn(1, 3))
}

func TestTheIDsOfAFinaleAreNeverEmpty(t *testing.T) {
	_, err := finales.BookIDOf("")
	require.ErrorIs(t, err, finales.ErrNoBook)
	_, err = finales.RoundIDOf("")
	require.ErrorIs(t, err, finales.ErrNoRound)

	book, err := finales.BookIDOf("season-0")
	require.NoError(t, err)
	assert.Equal(t, "season-0", book.String())
}
