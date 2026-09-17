package revert_player_usecase_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/revert_player_usecase"
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

func takenBy(book *inmemory_ledger_storage.Storage, scope string) int {
	n := 0
	book.Replay(func(taking ledger.Taking) {
		if taking.Scope == scope {
			n++
		}
	})
	return n
}

// The bot took 1-5 over whoever held them; somebody took 4 back and a bomb cleared 5.
func setup(t *testing.T) (*inmemory_ledger_storage.Storage, *stubMap) {
	t.Helper()

	book := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	tiles := &stubMap{owners: map[uint32]string{1: "il", 2: "", 3: "il", 4: "il", 5: "il"}}

	for tile := uint32(1); tile <= 5; tile++ {
		book.Append(ledger.Taking{Tile: tile, Scope: "9.9.9.9", Country: "ps", Previous: tiles.owners[tile]})
		tiles.owners[tile] = "ps"
	}
	book.Append(ledger.Taking{Tile: 4, Scope: "1.1.1.1", Country: "il", Previous: "ps"})
	tiles.owners[4] = "il"
	tiles.owners[5] = ""

	return book, tiles
}

func TestItGivesBackOnlyTheTilesStillWearingThePaintInBatches(t *testing.T) {
	book, tiles := setup(t)
	useCase := revert_player_usecase.New(book, tiles, clicks.Pacing{Batch: 2})

	out, err := useCase.Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9"})
	require.NoError(t, err)

	assert.Equal(t, revert_player_usecase.Out{Scope: "9.9.9.9", Touched: 5, Held: 3, Restored: 3}, out)
	assert.Equal(t, map[uint32]string{1: "il", 2: "", 3: "il", 4: "il", 5: ""}, tiles.owners)
	assert.Equal(t, []int{2, 1}, tiles.batches)
	assert.Zero(t, takenBy(book, "9.9.9.9"), "a reverted scope has nothing left to revert")
	assert.Equal(t, 1, takenBy(book, "1.1.1.1"))

	again, err := useCase.Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9"})
	require.NoError(t, err)
	assert.Equal(t, revert_player_usecase.Out{Scope: "9.9.9.9"}, again)
}

func TestADryRunCountsAndRestoresNothing(t *testing.T) {
	book, tiles := setup(t)
	useCase := revert_player_usecase.New(book, tiles, clicks.Pacing{Batch: 2})

	out, err := useCase.Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9", DryRun: true})
	require.NoError(t, err)

	assert.Equal(t, revert_player_usecase.Out{Scope: "9.9.9.9", Touched: 5, Held: 3}, out)
	assert.Empty(t, tiles.batches)
	assert.Equal(t, 5, takenBy(book, "9.9.9.9"))
}

func TestItGivesBackOnlyTheScopesLatestRunWhenTakesInterleave(t *testing.T) {
	book := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	tiles := &stubMap{owners: map[uint32]string{7: "il", 8: "il"}}

	take := func(tile uint32, scope, country string) {
		book.Append(ledger.Taking{Tile: tile, Scope: scope, Country: country, Previous: tiles.owners[tile]})
		tiles.owners[tile] = country
	}
	take(7, "9.9.9.9", "ps")
	take(7, "1.1.1.1", "de")
	take(7, "9.9.9.9", "ps")
	take(7, "9.9.9.9", "fr")
	take(8, "9.9.9.9", "ps")
	take(8, "1.1.1.1", "il")

	out, err := revert_player_usecase.New(book, tiles, clicks.Pacing{Batch: 10}).
		Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9"})
	require.NoError(t, err)

	assert.Equal(t, revert_player_usecase.Out{Scope: "9.9.9.9", Touched: 2, Held: 1, Restored: 1}, out)
	assert.Equal(t, map[uint32]string{7: "de", 8: "il"}, tiles.owners,
		"tile 7 goes back to the retake that broke the run, and tile 8 was taken back already")
}

func TestItRefusesWhatIsNotAScope(t *testing.T) {
	book, tiles := setup(t)

	_, err := revert_player_usecase.New(book, tiles, clicks.Pacing{Batch: 2}).Execute(t.Context(), revert_player_usecase.In{Scope: "bot"})
	require.ErrorIs(t, err, ledger.ErrInvalidScope)
}

func TestItStopsWhenTheContextEndsAndSaysHowFarItGot(t *testing.T) {
	book, tiles := setup(t)
	useCase := revert_player_usecase.New(book, tiles, clicks.Pacing{Batch: 2, Pause: time.Hour})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	out, err := useCase.Execute(ctx, revert_player_usecase.In{Scope: "9.9.9.9"})
	require.ErrorIs(t, err, context.Canceled)

	assert.Equal(t, 2, out.Restored, "the first batch went before the pause noticed")
	assert.Equal(t, 5, takenBy(book, "9.9.9.9"), "an interrupted revert can be run again")
}

func TestAStorageErrorIsReturned(t *testing.T) {
	book, tiles := setup(t)
	tiles.err = errors.New("code table full")

	_, err := revert_player_usecase.New(book, tiles, clicks.Pacing{Batch: 2}).Execute(t.Context(), revert_player_usecase.In{Scope: "9.9.9.9"})
	require.ErrorIs(t, err, tiles.err)
}

func TestAnAccountIsRevertedWhateverScopeItTookFrom(t *testing.T) {
	const guest = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"
	book := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	tiles := &stubMap{owners: map[uint32]string{1: "il", 2: "il", 3: "il"}}

	book.Append(ledger.Taking{Tile: 1, Scope: "campus", Account: guest, Country: "ps", Previous: "il"})
	book.Append(ledger.Taking{Tile: 2, Scope: "home", Account: guest, Country: "ps", Previous: "il"})
	book.Append(ledger.Taking{Tile: 3, Scope: "campus", Account: "a-classmate", Country: "ps", Previous: "il"})
	for tile := uint32(1); tile <= 3; tile++ {
		tiles.owners[tile] = "ps"
	}

	out, err := revert_player_usecase.New(book, tiles, clicks.Pacing{Batch: 10}).
		Execute(t.Context(), revert_player_usecase.In{Account: guest})
	require.NoError(t, err)

	assert.Equal(t, revert_player_usecase.Out{Account: guest, Touched: 2, Held: 2, Restored: 2}, out)
	assert.Equal(t, map[uint32]string{1: "il", 2: "il", 3: "ps"}, tiles.owners, "the classmate on the same scope keeps its tile")

	again, err := revert_player_usecase.New(book, tiles, clicks.Pacing{Batch: 10}).
		Execute(t.Context(), revert_player_usecase.In{Account: guest})
	require.NoError(t, err)
	assert.Zero(t, again.Touched, "the account's takes are forgotten")
	assert.Equal(t, 1, takenBy(book, "campus"), "and only the account's")
}
