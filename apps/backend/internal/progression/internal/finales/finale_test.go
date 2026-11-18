package finales_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/finales"
)

var (
	ada = accounts.AccountID{15: 1}
	bob = accounts.AccountID{15: 2}
)

func TestEachFighterIsRankedByTheTilesItTookAndATieSharesTheRank(t *testing.T) {
	carol := accounts.AccountID{15: 3}
	dan := accounts.AccountID{15: 4}

	fights, err := finales.Result{
		Fighters: []finales.Fighter{
			{Account: ada, Country: "fr", Tiles: 40},
			{Account: bob, Country: "de", Tiles: 90},
			{Account: carol, Country: "fr", Tiles: 40},
			{Account: dan, Country: "it", Tiles: 7},
		},
	}.Fights()

	require.NoError(t, err)
	ranks := map[accounts.AccountID]uint32{}
	for _, fight := range fights {
		ranks[fight.Account()] = fight.TakeSnapshot().TilesRank
	}
	assert.Equal(t, map[accounts.AccountID]uint32{bob: 1, ada: 2, carol: 2, dan: 4}, ranks)
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
