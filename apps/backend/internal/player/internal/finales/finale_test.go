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
		Fighters: []finales.Fighter{{Account: ada, Country: "de"}, {Account: bob, Country: "tv"}},
	}.Fights()

	require.NoError(t, err)
	require.Len(t, fights, 2)
	assert.Equal(t, finales.FightSnapshot{Account: ada, Season: 0, Country: "de", Rank: 1}, fights[0].TakeSnapshot(),
		"a tie shares the rank")
	assert.Equal(t, finales.FightSnapshot{Account: bob, Season: 0, Country: "tv", Rank: 0}, fights[1].TakeSnapshot())
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

func TestTheIDsOfAFinaleAreNeverEmpty(t *testing.T) {
	_, err := finales.BookIDOf("")
	require.ErrorIs(t, err, finales.ErrNoBook)
	_, err = finales.RoundIDOf("")
	require.ErrorIs(t, err, finales.ErrNoRound)

	book, err := finales.BookIDOf("season-0")
	require.NoError(t, err)
	assert.Equal(t, "season-0", book.String())
}
