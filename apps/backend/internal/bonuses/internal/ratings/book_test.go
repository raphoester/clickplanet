package ratings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/ratings"
)

func TestTheBookStartsFromWhatPostgresKept(t *testing.T) {
	kept := map[ratings.QuestionID]ratings.Rating{"capital:ee": ratings.RatingOf(1.5, 12, 4)}
	book := ratings.NewBook(kept)

	rating, ok := book.Rated("capital:ee")
	require.True(t, ok)
	assert.Equal(t, ratings.RatingOf(1.5, 12, 4), rating)

	_, ok = book.Rated("capital:lv")
	assert.False(t, ok, "a question no answer moved is at its prior, which is the bank's")

	delete(kept, "capital:ee")
	_, ok = book.Rated("capital:ee")
	assert.True(t, ok, "the book keeps its own copy")
}

func TestTheBookKeepsWhatAnAnswerCommitted(t *testing.T) {
	book := ratings.NewBook(nil)

	book.Keep("capital:ee", ratings.RatingOf(0.5, 1, 1))

	rating, ok := book.Rated("capital:ee")
	require.True(t, ok)
	assert.Equal(t, 1, rating.Answers())
}
