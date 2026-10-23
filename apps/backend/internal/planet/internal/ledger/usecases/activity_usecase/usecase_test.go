package activity_usecase_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/activity_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

type grid struct{}

func (grid) Land() []byte { return []byte{3, 0} }

func (grid) Heat(changes map[uint32]int) []uint32 {
	return []uint32{uint32(changes[1]), uint32(changes[2])} //nolint:gosec // counts of a test.
}

type territories map[string]uint32

func (t territories) Territories() map[string]uint32 { return t }

type audience bonuses.Audience

func (a audience) Audience() bonuses.Audience { return bonuses.Audience(a) }

func setup(t *testing.T) *activity_usecase.UseCase {
	t.Helper()

	book := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(),
		slog.New(slog.DiscardHandler))
	book.Append(ledger.Taking{Tile: 1, Scope: "A", Country: "fr", At: now.Add(-25 * time.Hour)})
	book.Append(ledger.Taking{Tile: 1, Scope: "B", Country: "de", Previous: "fr", At: now.Add(-2 * time.Hour)})
	book.Append(ledger.Taking{Tile: 2, Scope: "B", Country: "de", At: now.Add(-time.Hour)})

	return activity_usecase.New(book, grid{}, territories{"de": 2}, audience{Streams: 3, Entrants: 2},
		cptime.NewFixedClock(now))
}

func TestItReadsTheLastDayUnlessAskedAndDrawsWhatChangedHandsOnTheGrid(t *testing.T) {
	out, err := setup(t).Execute(t.Context(), activity_usecase.In{})
	require.NoError(t, err)

	assert.Equal(t, now.Add(-24*time.Hour).Add(15*time.Minute), out.Activity.Since, "96 slots of 15 minutes, the last one starting now")
	assert.Equal(t, 1, out.Activity.Players, "the take of 25 hours ago is out of the span")
	assert.Equal(t, map[string]ledger.Moves{"de": {Won: 2}, "fr": {Lost: 1}}, out.Activity.Moves)
	assert.Equal(t, []byte{3, 0}, out.Land)
	assert.Equal(t, []uint32{1, 1}, out.Heat)
	assert.Equal(t, map[string]uint32{"de": 2}, out.Held)
	assert.Equal(t, bonuses.Audience{Streams: 3, Entrants: 2}, out.Audience)
}

func TestALongerSpanReadsFurtherBack(t *testing.T) {
	out, err := setup(t).Execute(t.Context(), activity_usecase.In{Span: 72 * time.Hour})
	require.NoError(t, err)

	assert.Equal(t, 2, out.Activity.Players)
	assert.Equal(t, map[string]ledger.Moves{"de": {Won: 2}, "fr": {Won: 1, Lost: 1}}, out.Activity.Moves)
}

func TestASpanPastTheRetentionIsRefused(t *testing.T) {
	_, err := setup(t).Execute(t.Context(), activity_usecase.In{Span: 73 * time.Hour})

	assert.ErrorIs(t, err, ledger.ErrInvalidSpan)
}
