package count_take_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_round_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/count_take_usecase"
)

func utc(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
}

var (
	seasonZero = books.NewBook("season-0", utc(8, 0))
	day24      = books.Round{ID: "season-0/day-24", Book: seasonZero.ID, Times: 1, StartsAt: utc(30, 21)}
	battle     = books.Round{ID: "battle", Book: seasonZero.ID, Times: 3, StartsAt: utc(31, 21)}
	ada        = books.AccountID{15: 1}
)

func TestATakeCountsInEveryRoundOpenAtItsTimeOfABookOpenAtItsTime(t *testing.T) {
	shelf, rounds := inmemory_book_store.New(), inmemory_round_store.New()
	_, err := shelf.Open(t.Context(), seasonZero)
	require.NoError(t, err)
	for _, round := range []books.Round{day24, battle} {
		_, err := rounds.Open(t.Context(), round)
		require.NoError(t, err)
	}
	_, err = rounds.Score(t.Context(), day24.ScoredAt(utc(31, 21), books.Snapshot{Tiles: 1}))
	require.NoError(t, err)
	_, err = shelf.Close(t.Context(), seasonZero.ClosedAt(utc(31, 23), battle.ID))
	require.NoError(t, err)
	useCase := count_take_usecase.New(shelf, rounds, rounds)

	for position, at := range []time.Time{
		utc(31, 20), utc(31, 21), utc(31, 22), utc(31, 23), utc(30, 20),
	} {
		require.NoError(t, useCase.Execute(t.Context(),
			books.Take{Account: ada, Country: "fr", At: at, Position: uint64(position)})) //nolint:gosec // five takes.
	}

	assert.Equal(t, map[books.Country]uint64{"fr": 1}, rounds.CountedIn(day24.ID).Countries,
		"before its end, told after it was scored")
	assert.Equal(t, map[books.Country]uint64{"fr": 2}, rounds.CountedIn(battle.ID).Countries,
		"from its start to the book's end")
}
