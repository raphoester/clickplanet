package books_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

func TestAnIDIsOneToSixtyFourCharacters(t *testing.T) {
	for _, id := range []string{"", strings.Repeat("a", 65)} {
		_, err := books.ParseBookID(id)
		require.ErrorIs(t, err, books.ErrInvalidID)
		_, err = books.ParseRoundID(id)
		require.ErrorIs(t, err, books.ErrInvalidID)
		_, err = books.ParseGrantID(id)
		require.ErrorIs(t, err, books.ErrInvalidID)
	}
	id, err := books.ParseRoundID(strings.Repeat("a", 64))
	require.NoError(t, err)
	assert.Equal(t, books.RoundID(strings.Repeat("a", 64)), id)
}
