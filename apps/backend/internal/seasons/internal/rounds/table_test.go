package rounds_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
)

func TestTheTableIsByPointsThenRoundsWonThenTheFinale(t *testing.T) {
	table := rounds.TableOf([]rounds.Score{
		{Country: "es", Points: 40, RoundsWon: 1, FinalePoints: 54},
		{Country: "fr", Points: 50},
		{Country: "it", Points: 40, RoundsWon: 1, FinalePoints: 75},
		{Country: "de", Points: 40, RoundsWon: 2},
	})

	assert.Equal(t, []rounds.Placed{
		{Rank: 1, Score: rounds.Score{Country: "fr", Points: 50}},
		{Rank: 2, Score: rounds.Score{Country: "de", Points: 40, RoundsWon: 2}},
		{Rank: 3, Score: rounds.Score{Country: "it", Points: 40, RoundsWon: 1, FinalePoints: 75}},
		{Rank: 4, Score: rounds.Score{Country: "es", Points: 40, RoundsWon: 1, FinalePoints: 54}},
	}, table)
}

func TestCountriesLevelOnEverythingShareTheRank(t *testing.T) {
	table := rounds.TableOf([]rounds.Score{
		{Country: "it", Points: 18},
		{Country: "fr", Points: 25, RoundsWon: 1},
		{Country: "de", Points: 18},
	})

	assert.Equal(t, []rounds.Placed{
		{Rank: 1, Score: rounds.Score{Country: "fr", Points: 25, RoundsWon: 1}},
		{Rank: 2, Score: rounds.Score{Country: "de", Points: 18}},
		{Rank: 2, Score: rounds.Score{Country: "it", Points: 18}},
	}, table)
}
