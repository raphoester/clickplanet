package tell_streams_usecase_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/feed/usecases/tell_streams_usecase"
)

var (
	boot     = time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	openedAt = boot.Add(time.Minute)
	closedAt = openedAt.Add(time.Hour)
)

type fixture struct {
	store   *catalog.MemoryStore
	flags   <-chan catalog.Snapshot
	useCase *tell_streams_usecase.UseCase
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	store, board := catalog.NewMemoryStore(), feed.NewBoard(8)
	flags, leave := board.Attend()
	t.Cleanup(leave)
	return fixture{store: store, flags: flags, useCase: tell_streams_usecase.New(feed.NewsSince(boot), store, board)}
}

func TestAStreamHearsTheFlagAsTheCatalogHoldsItNow(t *testing.T) {
	f := newFixture(t)
	closed, err := catalog.OpenFlag("r-0000000001", "Rennes", openedAt).Closed(closedAt)
	require.NoError(t, err)
	require.NoError(t, f.store.Save(t.Context(), closed))

	require.NoError(t, f.useCase.Execute(t.Context(), tell_streams_usecase.In{Flag: "r-0000000001", DecidedAt: openedAt}))

	require.Len(t, f.flags, 1)
	assert.Equal(t, closed.TakeSnapshot(), <-f.flags, "the opening heard after the close tells the flag closed")
}

func TestAFactToldAgainAtTheBootIsNotNews(t *testing.T) {
	f := newFixture(t)
	require.NoError(t, f.store.Save(t.Context(), catalog.OpenFlag("r-0000000001", "Rennes", boot.Add(-time.Hour))))

	require.NoError(t, f.useCase.Execute(t.Context(),
		tell_streams_usecase.In{Flag: "r-0000000001", DecidedAt: boot.Add(-time.Hour)}))

	assert.Empty(t, f.flags)
}

func TestAFactOfAFlagNeverKeptIsRefused(t *testing.T) {
	f := newFixture(t)

	err := f.useCase.Execute(t.Context(), tell_streams_usecase.In{Flag: "r-0000000001", DecidedAt: openedAt})

	require.ErrorIs(t, err, catalog.ErrUnknownFlag)
	assert.Empty(t, f.flags)
}
