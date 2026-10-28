package books_test

import (
	"testing"

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

func TestABookIsRankedFromItsRoundsWithItsTiebreakAsTheThirdKey(t *testing.T) {
	battle := books.Round{ID: "battle", Book: "season-0", Times: 3}
	scored := []books.Scored{
		{Round: day, Results: []books.Result{{Country: "fr", Rank: 1, Points: 25}, {Country: "de", Rank: 2, Points: 18}}},
		{Round: battle, Results: []books.Result{{Country: "de", Rank: 1, Points: 75}, {Country: "fr", Rank: 2, Points: 54}, {Country: "it", Rank: 3, Points: 0}}},
	}

	closed := books.NewBook("season-0", utc(8, 0)).ClosedAt(utc(31, 23), "battle")

	assert.Equal(t, []books.Placed{
		{Rank: 1, Score: books.Score{Country: "de", Points: 93, RoundsWon: 1, TiebreakPoints: 75}},
		{Rank: 2, Score: books.Score{Country: "fr", Points: 79, RoundsWon: 1, TiebreakPoints: 54}},
	}, closed.Table(scored))
	assert.Equal(t, []books.Placed{
		{Rank: 1, Score: books.Score{Country: "de", Points: 93, RoundsWon: 1}},
		{Rank: 2, Score: books.Score{Country: "fr", Points: 79, RoundsWon: 1}},
	}, books.NewBook("season-0", utc(8, 0)).Table(scored), "no tiebreak while it runs")
}
