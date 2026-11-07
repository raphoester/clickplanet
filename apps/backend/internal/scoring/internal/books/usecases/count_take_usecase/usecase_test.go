package count_take_usecase_test

import (
	"context"
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
	bob        = books.AccountID{15: 2}
)

func seasonZeroClosed(t *testing.T) (*inmemory_book_store.Store, *inmemory_round_store.Store) {
	t.Helper()
	shelf, rounds := inmemory_book_store.New(), inmemory_round_store.New()
	_, err := shelf.Open(t.Context(), seasonZero)
	require.NoError(t, err)
	for _, round := range []books.Round{day24, battle} {
		_, err := rounds.Open(t.Context(), round, books.Start{})
		require.NoError(t, err)
	}
	_, err = rounds.Score(t.Context(), day24.ScoredAt(utc(31, 21), books.Start{}, books.Snapshot{Tiles: 1}))
	require.NoError(t, err)
	_, err = shelf.Close(t.Context(), seasonZero.ClosedAt(utc(31, 23), battle.ID))
	require.NoError(t, err)
	return shelf, rounds
}

func TestATakeCountsInEveryRoundOpenAtItsTimeOfABookOpenAtItsTime(t *testing.T) {
	shelf, rounds := seasonZeroClosed(t)
	useCase := count_take_usecase.New(shelf, rounds, rounds)

	for position, at := range []time.Time{
		utc(31, 20), utc(31, 21), utc(31, 22), utc(31, 23), utc(30, 20),
	} {
		require.NoError(t, useCase.Execute(t.Context(),
			[]books.Take{{Account: ada, Country: "fr", At: at, Position: uint64(position)}})) //nolint:gosec // five takes.
	}

	assert.Equal(t, map[books.Country]uint64{"fr": 1}, rounds.CountedIn(day24.ID).Countries,
		"before its end, told after it was scored")
	assert.Equal(t, map[books.Country]uint64{"fr": 2}, rounds.CountedIn(battle.ID).Countries,
		"from its start to the book's end")
}

type spy struct {
	*inmemory_round_store.Store

	reads, writes int
}

func (s *spy) Of(ctx context.Context, of []books.BookID) ([]books.Round, error) {
	s.reads++
	return s.Store.Of(ctx, of) //nolint:wrapcheck // a spy.
}

func (s *spy) Count(ctx context.Context, held map[books.RoundID][]books.Take) error {
	s.writes++
	return s.Store.Count(ctx, held) //nolint:wrapcheck // a spy.
}

func TestABatchCountsEachTakeInItsOwnRoundsInOneWrite(t *testing.T) {
	shelf, rounds := seasonZeroClosed(t)
	counted := &spy{Store: rounds}

	require.NoError(t, count_take_usecase.New(shelf, counted, counted).Execute(t.Context(), []books.Take{
		{Account: ada, Country: "fr", At: utc(30, 20), Position: 1},
		{Account: ada, Country: "fr", At: utc(31, 20), Position: 2, Index: 0},
		{Account: ada, Country: "fr", At: utc(31, 20), Position: 2, Index: 1},
		{Account: bob, Country: "de", At: utc(31, 22), Position: 3},
		{Account: ada, Country: "fr", At: utc(31, 22), Position: 4},
		{Account: bob, Country: "de", At: utc(31, 23), Position: 5},
	}))

	assert.Equal(t, 1, counted.reads)
	assert.Equal(t, 1, counted.writes)
	assert.Equal(t, books.Counted{
		Countries: map[books.Country]uint64{"fr": 2},
		Players:   map[books.AccountID]map[books.Country]uint64{ada: {"fr": 2}},
	}, rounds.CountedIn(day24.ID), "the enclosure, and not the take before the day")
	assert.Equal(t, books.Counted{
		Countries: map[books.Country]uint64{"fr": 1, "de": 1},
		Players:   map[books.AccountID]map[books.Country]uint64{ada: {"fr": 1}, bob: {"de": 1}},
	}, rounds.CountedIn(battle.ID), "from its start, and not past the book's end")
}

func TestABatchNoBookHoldsCountsNothing(t *testing.T) {
	shelf, rounds := seasonZeroClosed(t)
	counted := &spy{Store: rounds}

	require.NoError(t, count_take_usecase.New(shelf, counted, counted).Execute(t.Context(), []books.Take{
		{Account: ada, Country: "fr", At: utc(31, 23), Position: 1},
		{Account: ada, Country: "fr", At: utc(7, 23), Position: 2},
	}))

	assert.Zero(t, counted.reads)
	assert.Zero(t, counted.writes)
}
