package rounds_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
)

var finale = rounds.Round{EndsAt: utc(31, 23, 0, 0), Finale: true}

func TestASeasonScoresTheRoundsItWonPlacesIn(t *testing.T) {
	scores := rounds.ScoresOf([]rounds.Closed{
		{Round: day, Results: []rounds.Result{{Country: "fr", Rank: 1, Points: 25}, {Country: "de", Rank: 2, Points: 18}}},
		{Round: finale, Results: []rounds.Result{{Country: "de", Rank: 1, Points: 75}, {Country: "fr", Rank: 2, Points: 54}}},
	})

	assert.ElementsMatch(t, []rounds.Score{
		{Country: "fr", Points: 79, RoundsWon: 1, FinalePoints: 54},
		{Country: "de", Points: 93, RoundsWon: 1, FinalePoints: 75},
	}, scores)
}

func TestACountryThatScoredNothingIsNotInTheSeason(t *testing.T) {
	scores := rounds.ScoresOf([]rounds.Closed{
		{Round: day, Results: []rounds.Result{{Country: "fr", Rank: 1, Points: 25}, {Country: "nz", Rank: 11}}},
	})

	assert.Equal(t, []rounds.Score{{Country: "fr", Points: 25, RoundsWon: 1}}, scores)
}

func TestTheFinaleEndsTheSeasonWithItsTable(t *testing.T) {
	earlier := []rounds.Closed{
		{Round: day, Results: []rounds.Result{{Country: "fr", Rank: 1, Points: 25}, {Country: "de", Rank: 2, Points: 18}}},
	}
	closing := rounds.Closed{Round: finale, Number: 2, Results: []rounds.Result{{Country: "de", Rank: 1, Points: 75}}}

	end, ended := closing.SeasonEnd(earlier)

	require.True(t, ended)
	assert.Equal(t, rounds.SeasonEnd{
		Season:  0,
		EndedAt: finale.EndsAt,
		Table: []rounds.Placed{
			{Rank: 1, Score: rounds.Score{Country: "de", Points: 93, RoundsWon: 1, FinalePoints: 75}},
			{Rank: 2, Score: rounds.Score{Country: "fr", Points: 25, RoundsWon: 1}},
		},
	}, end)
}

func TestADayEndsNoSeason(t *testing.T) {
	_, ended := rounds.Closed{Round: day, Number: 1}.SeasonEnd(nil)

	assert.False(t, ended)
}
