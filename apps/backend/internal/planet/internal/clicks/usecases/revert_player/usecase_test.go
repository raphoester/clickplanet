package revert_player_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/pacing"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/revert_player"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type stubMap struct {
	owners  map[uint32]string
	batches []int
	err     error
}

func (m *stubMap) Owner(tile uint32) (string, bool) { return m.owners[tile], true }

func (m *stubMap) Restore(_ context.Context, restorations []clicks.Restoration) (int, error) {
	m.batches = append(m.batches, len(restorations))
	if m.err != nil {
		return 0, m.err
	}

	restored := 0
	for _, r := range restorations {
		if m.owners[r.Tile] == r.From {
			m.owners[r.Tile] = r.To
			restored++
		}
	}
	return restored, nil
}

// The bot took 1-5 over whoever held them; somebody took 4 back and a bomb cleared 5.
func setup(t *testing.T) (*ledger.Ledger, *stubMap) {
	t.Helper()

	book := ledger.New(ledger.Config{}, cptime.NewFixedClock(time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)))
	tiles := &stubMap{owners: map[uint32]string{1: "il", 2: "", 3: "il", 4: "il", 5: "il"}}

	for tile := uint32(1); tile <= 5; tile++ {
		book.Record(tile, "9.9.9.9", tiles.owners[tile], "ps")
		tiles.owners[tile] = "ps"
	}
	book.Record(4, "1.1.1.1", "ps", "il")
	tiles.owners[4] = "il"
	tiles.owners[5] = ""

	return book, tiles
}

func TestItGivesBackOnlyTheTilesStillWearingThePaintInBatches(t *testing.T) {
	book, tiles := setup(t)
	useCase := revert_player.New(book, tiles, pacing.Pacing{Batch: 2})

	out, err := useCase.Execute(t.Context(), revert_player.In{Scope: "9.9.9.9"})
	require.NoError(t, err)

	assert.Equal(t, revert_player.Out{Scope: "9.9.9.9", Touched: 4, Held: 3, Restored: 3}, out)
	assert.Equal(t, map[uint32]string{1: "il", 2: "", 3: "il", 4: "il", 5: ""}, tiles.owners)
	assert.Equal(t, []int{2, 1}, tiles.batches)
	assert.Empty(t, book.TakenBy("9.9.9.9"), "a reverted scope has nothing left to revert")
	assert.Len(t, book.TakenBy("1.1.1.1"), 1)
}

func TestADryRunCountsAndRestoresNothing(t *testing.T) {
	book, tiles := setup(t)
	useCase := revert_player.New(book, tiles, pacing.Pacing{Batch: 2})

	out, err := useCase.Execute(t.Context(), revert_player.In{Scope: "9.9.9.9", DryRun: true})
	require.NoError(t, err)

	assert.Equal(t, revert_player.Out{Scope: "9.9.9.9", Touched: 4, Held: 3}, out)
	assert.Empty(t, tiles.batches)
	assert.Len(t, book.TakenBy("9.9.9.9"), 4)
}

func TestItRefusesWhatIsNotAScope(t *testing.T) {
	book, tiles := setup(t)

	_, err := revert_player.New(book, tiles, pacing.Pacing{Batch: 2}).Execute(t.Context(), revert_player.In{Scope: "bot"})
	require.ErrorIs(t, err, clicks.ErrInvalidScope)
}

func TestItStopsWhenTheContextEndsAndSaysHowFarItGot(t *testing.T) {
	book, tiles := setup(t)
	useCase := revert_player.New(book, tiles, pacing.Pacing{Batch: 2, Pause: time.Hour})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	out, err := useCase.Execute(ctx, revert_player.In{Scope: "9.9.9.9"})
	require.ErrorIs(t, err, context.Canceled)

	assert.Equal(t, 2, out.Restored, "the first batch went before the pause noticed")
	assert.Len(t, book.TakenBy("9.9.9.9"), 4, "an interrupted revert can be run again")
}

func TestAStorageErrorIsReturned(t *testing.T) {
	book, tiles := setup(t)
	tiles.err = errors.New("code table full")

	_, err := revert_player.New(book, tiles, pacing.Pacing{Batch: 2}).Execute(t.Context(), revert_player.In{Scope: "9.9.9.9"})
	require.ErrorIs(t, err, tiles.err)
}
