package books_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

func TestTheTableIsByPointsThenRoundsWonThenTheTiebreak(t *testing.T) {
	table := books.TableOf([]books.Score{
		{Country: "es", Points: 40, RoundsWon: 1, TiebreakPoints: 54},
		{Country: "fr", Points: 50},
		{Country: "it", Points: 40, RoundsWon: 1, TiebreakPoints: 75},
		{Country: "de", Points: 40, RoundsWon: 2},
	})

	assert.Equal(t, []books.Placed{
		{Rank: 1, Score: books.Score{Country: "fr", Points: 50}},
		{Rank: 2, Score: books.Score{Country: "de", Points: 40, RoundsWon: 2}},
		{Rank: 3, Score: books.Score{Country: "it", Points: 40, RoundsWon: 1, TiebreakPoints: 75}},
		{Rank: 4, Score: books.Score{Country: "es", Points: 40, RoundsWon: 1, TiebreakPoints: 54}},
	}, table)
}

func TestCountriesLevelOnEverythingShareTheRank(t *testing.T) {
	table := books.TableOf([]books.Score{
		{Country: "it", Points: 18},
		{Country: "fr", Points: 25, RoundsWon: 1},
		{Country: "de", Points: 18},
	})

	assert.Equal(t, []books.Placed{
		{Rank: 1, Score: books.Score{Country: "fr", Points: 25, RoundsWon: 1}},
		{Rank: 2, Score: books.Score{Country: "de", Points: 18}},
		{Rank: 2, Score: books.Score{Country: "it", Points: 18}},
	}, table)
}

func TestAGrantAddsPointsToTheBooksOpenAtItsTimeAndWinsNoRound(t *testing.T) {
	scored := []books.Scored{{Round: day, Results: []books.Result{{Country: "fr", Rank: 1, Points: 25}}}}
	grants := []books.Grant{
		{ID: "before", Country: "fr", Points: 3, At: utc(8, 0).Add(-time.Second)},
		{ID: "fr", Country: "fr", Points: 2, At: utc(9, 0)},
		{ID: "it", Country: "it", Points: 1, At: utc(9, 0)},
		{ID: "after", Country: "it", Points: 3, At: utc(31, 23)},
	}

	closed := books.NewBook("season-0", utc(8, 0)).ClosedAt(utc(31, 23), day.ID)

	assert.Equal(t, []books.Placed{
		{Rank: 1, Score: books.Score{Country: "fr", Points: 27, RoundsWon: 1, TiebreakPoints: 25}},
		{Rank: 2, Score: books.Score{Country: "it", Points: 1}},
	}, closed.Table(scored, grants))
}
