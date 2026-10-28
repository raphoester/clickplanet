package record_take_usecase_test

import (
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

	require.NoError(t, useCase.Execute(t.Context(), standings.Take{Account: ada, Country: "fr", At: seasonZeroEnds.Add(-time.Hour)}))
	require.NoError(t, useCase.Execute(t.Context(),
		standings.Take{Account: ada, Country: "fr", At: seasonZeroEnds.Add(-time.Minute), Position: 1}))

	two := standings.Tally{Main: "fr", Tiles: map[standings.Country]uint64{"fr": 2}, TakesBelow: standings.TakeMark{Position: 1, Index: 1}}
	assert.Equal(t, two, store.Tally("season-0", ada))
	assert.Equal(t, two, store.Tally("lifetime", ada))
}

func TestATakeOutsideABooksTimeCountsNothingInIt(t *testing.T) {
	store := inmemory_contribution_store.New()
	useCase := record_take_usecase.New(twoBooks(t), store)

	require.NoError(t, useCase.Execute(t.Context(), standings.Take{Account: ada, Country: "fr", At: seasonZeroEnds}))
	require.NoError(t, useCase.Execute(t.Context(),
		standings.Take{Account: ada, Country: "de", At: seasonZeroStarts.Add(-time.Minute), Position: 1}))

	assert.True(t, store.Tally("season-0", ada).Empty(), "at its end, and before its start")
	assert.Equal(t, standings.Tally{
		Main: "fr", Tiles: map[standings.Country]uint64{"fr": 1, "de": 1},
		TakesBelow: standings.TakeMark{Position: 1, Index: 1},
	},
		store.Tally("lifetime", ada))
}

func TestAFailureToCountIsAnError(t *testing.T) {
	store := inmemory_contribution_store.New()
	refused := errors.New("the database is down")
	store.FailWith(refused)

	err := record_take_usecase.New(twoBooks(t), store).Execute(t.Context(),
		standings.Take{Account: ada, Country: "fr", At: seasonZeroEnds.Add(-time.Hour)})

	assert.ErrorIs(t, err, refused)
}
