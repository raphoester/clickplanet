package replay_usecase_test

import (
	"encoding/binary"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/replay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

type board uint32

func (b board) MaxIndex() uint32 { return uint32(b) }

type tiles struct {
	owners []string
	asked  [2]uint32
	err    error
}

func (t *tiles) StateBatchDense(start uint32, end uint32) (clicks.DenseBatch, error) {
	t.asked = [2]uint32{start, end}
	batch := clicks.DenseBatch{Start: start, Codes: []string{""}}
	for _, owner := range t.owners {
		code := uint16(len(batch.Codes))
		batch.Codes = append(batch.Codes, owner)
		batch.Tiles = binary.LittleEndian.AppendUint16(batch.Tiles, code)
	}
	return batch, t.err
}

func setup(t *testing.T, owners ...string) (*inmemory_ledger_storage.Storage, *tiles, *replay_usecase.UseCase) {
	t.Helper()
	book, now, useCase, _ := setupWithResets(t, owners...)
	return book, now, useCase
}

func setupWithResets(t *testing.T, owners ...string) (*inmemory_ledger_storage.Storage, *tiles, *replay_usecase.UseCase, *resets.MemoryStore) {
	t.Helper()
	book := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{},
		inmemory_ledger_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler))
	now := &tiles{owners: owners}
	kept := resets.NewMemoryStore()
	return book, now, replay_usecase.New(book, kept, board(len(owners)), now, cptime.NewFixedClock(start.Add(time.Hour))), kept
}

func reset(t *testing.T, kept *resets.MemoryStore, id string, at time.Time) {
	t.Helper()
	applied, err := resets.New(id, at)
	require.NoError(t, err)
	_, err = kept.Keep(t.Context(), applied)
	require.NoError(t, err)
}

func TestAReplayOverAResetStartsAtItOnTheBlankMap(t *testing.T) {
	book, _, useCase, kept := setupWithResets(t, "de")
	book.Append(ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start})
	reset(t, kept, "season-1", start.Add(time.Minute))
	book.Append(ledger.Taking{Tile: 1, Scope: "b", Country: "de", Previous: "", At: start.Add(2 * time.Minute)})

	out, err := useCase.Execute(t.Context(), replay_usecase.In{Since: start})
	require.NoError(t, err)

	assert.Equal(t, start.Add(time.Minute), out.Since)
	assert.Empty(t, out.Opening.Codes[binary.LittleEndian.Uint16(out.Opening.Tiles)], "the tile was blank at the reset")
	require.Len(t, out.Scenes, 1)
	assert.Equal(t, "de", out.Scenes[0].Change.Value)
}

func TestAReplayThatEndedBeforeAResetIsRefused(t *testing.T) {
	_, _, useCase, kept := setupWithResets(t, "de")
	reset(t, kept, "season-1", start.Add(30*time.Minute))

	_, err := useCase.Execute(t.Context(), replay_usecase.In{Since: start, Until: start.Add(10 * time.Minute)})

	require.ErrorIs(t, err, replay_usecase.ErrBeforeReset)
}

func TestAResetBeforeTheWindowChangesNothing(t *testing.T) {
	_, _, useCase, kept := setupWithResets(t, "de")
	reset(t, kept, "season-1", start.Add(-time.Hour))

	out, err := useCase.Execute(t.Context(), replay_usecase.In{Since: start})

	require.NoError(t, err)
	assert.Equal(t, start, out.Since)
}

func TestAReplayOpensOnTheMapAtSinceAndShowsWhatCameAfter(t *testing.T) {
	book, now, useCase := setup(t, "de")
	book.Append(ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start})
	book.Append(ledger.Taking{Tile: 1, Scope: "b", Country: "de", Previous: "fr", At: start.Add(time.Minute)})

	out, err := useCase.Execute(t.Context(), replay_usecase.In{Since: start})
	require.NoError(t, err)

	assert.Equal(t, start, out.Since)
	assert.Equal(t, start.Add(time.Hour), out.Until, "an unset until is now")
	assert.Equal(t, [2]uint32{1, 1}, now.asked, "the whole map, from the first tile")
	assert.Equal(t, uint16(len(out.Opening.Codes)-1), binary.LittleEndian.Uint16(out.Opening.Tiles))
	assert.Equal(t, "it", out.Opening.Codes[len(out.Opening.Codes)-1])
	require.Len(t, out.Scenes, 2)
	assert.Equal(t, "fr", out.Scenes[0].Change.Value)
	assert.Equal(t, "de", out.Scenes[1].Change.Value)
}

func TestAReplayStopsAtUntil(t *testing.T) {
	book, _, useCase := setup(t, "de")
	book.Append(ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start})
	book.Append(ledger.Taking{Tile: 1, Scope: "b", Country: "de", Previous: "fr", At: start.Add(time.Minute)})

	out, err := useCase.Execute(t.Context(), replay_usecase.In{Since: start, Until: start.Add(time.Second)})
	require.NoError(t, err)

	require.Len(t, out.Scenes, 1)
	assert.Equal(t, "fr", out.Scenes[0].Change.Value)
}

func TestAReplayNeedsASinceBeforeItsUntil(t *testing.T) {
	_, _, useCase := setup(t, "de")

	for _, in := range []replay_usecase.In{
		{},
		{Since: start, Until: start},
		{Since: start.Add(2 * time.Hour)},
	} {
		_, err := useCase.Execute(t.Context(), in)
		assert.ErrorIs(t, err, replay_usecase.ErrInvalidWindow)
	}
}

func TestAMapThatCannotBeReadFailsTheReplay(t *testing.T) {
	_, now, useCase := setup(t, "de")
	now.err = errors.New("no map")

	_, err := useCase.Execute(t.Context(), replay_usecase.In{Since: start})

	require.ErrorIs(t, err, now.err)
}
