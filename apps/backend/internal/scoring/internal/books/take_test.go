package books_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

func taken(marks ...books.TakeMark) []books.Take {
	takes := make([]books.Take, 0, len(marks))
	for _, mark := range marks {
		takes = append(takes, books.Take{Country: "fr", Position: mark.Position, Index: mark.Index})
	}
	return takes
}

func TestATakeToldTwiceInOneBatchIsKeptOnce(t *testing.T) {
	uncounted, past := books.TakeMark{}.Uncounted(taken(
		books.TakeMark{Position: 1}, books.TakeMark{Position: 2}, books.TakeMark{Position: 2}, books.TakeMark{Position: 1},
	))

	assert.Equal(t, taken(books.TakeMark{Position: 1}, books.TakeMark{Position: 2}), uncounted)
	assert.Equal(t, books.TakeMark{Position: 2, Index: 1}, past)
}
