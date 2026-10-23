package excerpt_usecase_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/excerpt_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

func setup(t *testing.T) *excerpt_usecase.UseCase {
	t.Helper()

	book := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(),
		slog.New(slog.DiscardHandler))
	book.Append(ledger.Taking{Tile: 1, Scope: "A", Country: "fr", At: now.Add(-25 * time.Hour)})
	book.Append(ledger.Taking{Tile: 1, Scope: "B", Account: "acc", Country: "de", Previous: "fr", At: now.Add(-2 * time.Hour)})

	return excerpt_usecase.New(book, cptime.NewFixedClock(now))
}

func TestAnUnsetUntilIsNowAndOnlyTheWindowIsRead(t *testing.T) {
	out, err := setup(t).Execute(t.Context(), excerpt_usecase.In{Since: now.Add(-24 * time.Hour)})
	require.NoError(t, err)

	assert.Equal(t, excerpt_usecase.Out{
		Since: now.Add(-24 * time.Hour),
		Until: now,
		Lines: []ledger.Line{{
			At: now.Add(-2 * time.Hour), By: ledger.Caller{Account: "acc"},
			Update: clicks.TileUpdate{Tile: 1, Value: "de", Previous: "fr", Clicked: true},
		}},
	}, out)
}

func TestAWindowOfNothingOrPastTheLedgersMemoryIsRefused(t *testing.T) {
	for _, in := range []excerpt_usecase.In{
		{},
		{Since: now},
		{Since: now.Add(-time.Hour), Until: now.Add(-2 * time.Hour)},
		{Since: now.Add(-72*time.Hour - time.Second)},
	} {
		_, err := setup(t).Execute(t.Context(), in)
		assert.ErrorIs(t, err, excerpt_usecase.ErrInvalidWindow, in)
	}
}
