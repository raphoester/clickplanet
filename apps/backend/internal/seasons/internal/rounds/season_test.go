package rounds_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
)

var finale = rounds.Round{EndsAt: utc(31, 23, 0, 0), Finale: true, Own: true, StartsAt: utc(31, 21, 0, 0), Times: 3}

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

func TestASeasonEndsAtItsEndWithTheTableOfEveryRoundItClosed(t *testing.T) {
	season := calendar.Season{Number: 0, LastDayEndsAt: utc(31, 21, 0, 0), EndsAt: utc(31, 23, 0, 0)}

	end := rounds.SeasonEndOf(season, []rounds.Closed{
		{Round: day, Results: []rounds.Result{{Country: "fr", Rank: 1, Points: 25}, {Country: "de", Rank: 2, Points: 18}}},
		{Round: finale, Number: 1, Results: []rounds.Result{{Country: "de", Rank: 1, Points: 75}}},
	})

	assert.Equal(t, rounds.SeasonEnd{
		Season:  0,
		EndedAt: season.EndsAt,
		Table: []rounds.Placed{
			{Rank: 1, Score: rounds.Score{Country: "de", Points: 93, RoundsWon: 1, FinalePoints: 75}},
			{Rank: 2, Score: rounds.Score{Country: "fr", Points: 25, RoundsWon: 1}},
		},
	}, end)
}

func TestASeasonWithNoRoundAtItsEndEndsWithNoFinalePoints(t *testing.T) {
	season := calendar.Season{Number: 0, LastDayEndsAt: utc(31, 21, 0, 0), EndsAt: utc(31, 23, 0, 0)}

	end := rounds.SeasonEndOf(season, []rounds.Closed{
		{Round: day, Results: []rounds.Result{{Country: "fr", Rank: 1, Points: 25}}},
	})

	assert.Equal(t, []rounds.Placed{{Rank: 1, Score: rounds.Score{Country: "fr", Points: 25, RoundsWon: 1}}}, end.Table)
	assert.Equal(t, season.EndsAt, end.EndedAt)
}
