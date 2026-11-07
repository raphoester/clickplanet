package record_take_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/usecases/record_take_usecase"
)

var (
	seasonZeroStarts = time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	seasonZeroEnds   = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	ada              = standings.AccountID{0: 1, 15: 1}
	bob              = standings.AccountID{0: 1, 15: 2}
)

func twoBooks(t *testing.T) *inmemory_book_store.Store {
	t.Helper()
	store := inmemory_book_store.New()
	_, err := store.Open(t.Context(), books.NewBook("season-0", seasonZeroStarts))
	require.NoError(t, err)
	_, err = store.Close(t.Context(), books.NewBook("season-0", seasonZeroStarts).ClosedAt(seasonZeroEnds, ""))
	require.NoError(t, err)
	_, err = store.Open(t.Context(), books.NewBook("lifetime", seasonZeroStarts.Add(-time.Hour)))
	require.NoError(t, err)
	return store
}

func TestATakeCountsOneTileForItsFlagInEveryBookOpenAtItsTime(t *testing.T) {
	store := inmemory_contribution_store.New()
	useCase := record_take_usecase.New(twoBooks(t), store)

	require.NoError(t, useCase.Execute(t.Context(), []standings.Take{
		{Account: ada, Country: "fr", At: seasonZeroEnds.Add(-time.Hour)},
		{Account: ada, Country: "fr", At: seasonZeroEnds.Add(-time.Minute), Position: 1},
	}))

	two := standings.Tally{Main: "fr", Tiles: map[standings.Country]uint64{"fr": 2}, TakesBelow: standings.TakeMark{Position: 1, Index: 1}}
	assert.Equal(t, two, store.Tally("season-0", ada))
	assert.Equal(t, two, store.Tally("lifetime", ada))
}

func TestATakeOutsideABooksTimeCountsNothingInIt(t *testing.T) {
	store := inmemory_contribution_store.New()
	useCase := record_take_usecase.New(twoBooks(t), store)

	require.NoError(t, useCase.Execute(t.Context(), []standings.Take{
		{Account: ada, Country: "fr", At: seasonZeroEnds},
		{Account: ada, Country: "de", At: seasonZeroStarts.Add(-time.Minute), Position: 1},
	}))

	assert.True(t, store.Tally("season-0", ada).Empty(), "at its end, and before its start")
	assert.Equal(t, standings.Tally{
		Main: "fr", Tiles: map[standings.Country]uint64{"fr": 1, "de": 1},
		TakesBelow: standings.TakeMark{Position: 1, Index: 1},
	},
		store.Tally("lifetime", ada))
}

type recording struct {
	*inmemory_contribution_store.Store

	writes int
	books  int
}

func (r *recording) All(context.Context) ([]books.Book, error) {
	r.books++
	return []books.Book{books.NewBook("season-0", seasonZeroStarts), books.NewBook("lifetime", seasonZeroStarts)}, nil
}

func (r *recording) RecordTakes(ctx context.Context, book books.BookID, account standings.AccountID, takes []standings.Take) error {
	r.writes++
	return r.Store.RecordTakes(ctx, book, account, takes) //nolint:wrapcheck // a spy.
}

func TestABatchReadsTheBooksOnceAndWritesOncePerBookAndAccount(t *testing.T) {
	spy := &recording{Store: inmemory_contribution_store.New()}
	at := seasonZeroEnds.Add(-time.Hour)

	require.NoError(t, record_take_usecase.New(spy, spy).Execute(t.Context(), []standings.Take{
		{Account: ada, Country: "fr", At: at, Position: 1},
		{Account: bob, Country: "de", At: at, Position: 2},
		{Account: ada, Country: "it", At: at, Position: 3, Index: 0},
		{Account: ada, Country: "it", At: at, Position: 3, Index: 1},
	}))

	assert.Equal(t, 1, spy.books)
	assert.Equal(t, 4, spy.writes, "two books, two accounts")
	for _, book := range []books.BookID{"season-0", "lifetime"} {
		assert.Equal(t, standings.Tally{
			Main: "it", Tiles: map[standings.Country]uint64{"fr": 1, "it": 2}, TakesBelow: standings.TakeMark{Position: 3, Index: 2},
		}, spy.Tally(book, ada))
		assert.Equal(t, standings.Tally{
			Main: "de", Tiles: map[standings.Country]uint64{"de": 1}, TakesBelow: standings.TakeMark{Position: 2, Index: 1},
		}, spy.Tally(book, bob))
	}
}

func TestAFailureToCountIsAnError(t *testing.T) {
	store := inmemory_contribution_store.New()
	refused := errors.New("the database is down")
	store.FailWith(refused)

	err := record_take_usecase.New(twoBooks(t), store).Execute(t.Context(),
		[]standings.Take{{Account: ada, Country: "fr", At: seasonZeroEnds.Add(-time.Hour)}})

	assert.ErrorIs(t, err, refused)
}
