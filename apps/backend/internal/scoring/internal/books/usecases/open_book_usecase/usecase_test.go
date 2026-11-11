package open_book_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/inmemory_book_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books/usecases/open_book_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

func utc(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
}

func TestABookOpenedAgainKeepsItsFirstStart(t *testing.T) {
	shelf := inmemory_book_store.New()
	useCase := open_book_usecase.New(cpatomic.FakeTransactioner{}, shelf)

	opened, err := useCase.Execute(t.Context(), open_book_usecase.Command{Book: "season-0", StartsAt: utc(8, 0)})
	require.NoError(t, err)
	assert.True(t, opened)
	opened, err = useCase.Execute(t.Context(), open_book_usecase.Command{Book: "season-0", StartsAt: utc(9, 0)})
	require.NoError(t, err)
	assert.False(t, opened)

	book, _, err := shelf.Find(t.Context(), "season-0")
	require.NoError(t, err)
	assert.Equal(t, books.NewBook("season-0", utc(8, 0)), book)
}

func TestABookACloseMissedNeverOpens(t *testing.T) {
	shelf := inmemory_book_store.New()
	require.NoError(t, shelf.KeepMissed(t.Context(), books.NewMissedBook("season-0", utc(31, 23))))

	_, err := open_book_usecase.New(cpatomic.FakeTransactioner{}, shelf).
		Execute(t.Context(), open_book_usecase.Command{Book: "season-0", StartsAt: utc(8, 0)})

	require.ErrorIs(t, err, books.ErrRefused)
	require.ErrorIs(t, err, books.ErrMissed)
	_, found, err := shelf.Find(t.Context(), "season-0")
	require.NoError(t, err)
	assert.False(t, found)
}
