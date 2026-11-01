package podiums_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/podiums"
)

var (
	ada = players.AccountID{15: 1}
	bob = players.AccountID{15: 2}
)

func TestThePodiumIsTheBoardsFirstThreeRanks(t *testing.T) {
	places := podiums.Board{Season: 0, Entries: []podiums.Entry{
		{Account: ada, Rank: 3}, {Account: bob, Rank: 4}, {Account: players.AccountID{15: 3}, Rank: 0},
	}}.Places()

	require.Len(t, places, 1)
	assert.Equal(t, podiums.PlaceSnapshot{Account: ada, Season: 0, Rank: 3}, places[0].TakeSnapshot())
}

func TestAPlaceOffThePodiumIsRefused(t *testing.T) {
	for _, rank := range []uint32{0, 4} {
		_, err := podiums.PlaceSnapshot{Account: ada, Rank: rank}.Restore()

		assert.ErrorIs(t, err, podiums.ErrOffThePodium, "rank %d", rank)
	}
}

func TestPlacesSayWhichStepOfWhichSeason(t *testing.T) {
	place, err := podiums.PlaceSnapshot{Account: ada, Season: 0, Rank: 2}.Restore()
	require.NoError(t, err)
	places := podiums.Places{place}

	assert.True(t, places.RankedIn(0, 2))
	assert.False(t, places.RankedIn(0, 1))
	assert.False(t, places.RankedIn(1, 2))
}
