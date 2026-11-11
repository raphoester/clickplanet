package replay_usecase_test

import (
	"context"
	"encoding/binary"
	"errors"
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/replay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

type tiles struct {
	owners []string
	kept   []string
	asked  [2]uint32
	err    error
}

func batchOf(start uint32, owners []string) clicks.DenseBatch {
	batch := clicks.DenseBatch{Start: start, Codes: []string{""}}
	for _, owner := range owners {
		code := uint16(len(batch.Codes))
		batch.Codes = append(batch.Codes, owner)
		batch.Tiles = binary.LittleEndian.AppendUint16(batch.Tiles, code)
	}
	return batch
}

func (t *tiles) StateBatchDense(start uint32, end uint32) (clicks.DenseBatch, error) {
	t.asked = [2]uint32{start, end}
	return batchOf(start, t.owners), t.err
}

func (t *tiles) Kept(_ context.Context, kept clicks.MapID) (clicks.DenseBatch, error) {
	if kept != clicks.World {
		return clicks.DenseBatch{}, errors.New("only the world's map is kept aside")
	}
	return batchOf(1, t.kept), t.err
}

type harness struct {
	book    *inmemory_ledger_storage.Storage
	now     *tiles
	resets  *resets.MemoryStore
	arenas  *arenas.MemoryStore
	useCase *replay_usecase.UseCase
}

func setup(t *testing.T, owners ...string) harness {
	t.Helper()
	h := harness{
		book: inmemory_ledger_storage.New(inmemory_ledger_storage.Config{},
			inmemory_ledger_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler)),
		now: &tiles{owners: owners}, resets: resets.NewMemoryStore(), arenas: arenas.NewMemoryStore(),
	}
	h.useCase = replay_usecase.New(h.book, h.resets, h.arenas, h.now, cptime.NewFixedClock(start.Add(time.Hour)))
	return h
}

func reset(t *testing.T, kept *resets.MemoryStore, id string, at time.Time) {
	t.Helper()
	applied, err := resets.New(id, at)
	require.NoError(t, err)
	_, err = kept.Keep(t.Context(), applied)
	require.NoError(t, err)
}

func TestAReplayOverAResetStartsAtItOnTheBlankMap(t *testing.T) {
	h := setup(t, "de")
	h.book.Append(ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start})
	reset(t, h.resets, "season-1", start.Add(time.Minute))
	h.book.Append(ledger.Taking{Tile: 1, Scope: "b", Country: "de", Previous: "", At: start.Add(2 * time.Minute)})

	out, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start})
	require.NoError(t, err)

	assert.Equal(t, start.Add(time.Minute), out.Since)
	assert.Empty(t, out.Opening.Codes[binary.LittleEndian.Uint16(out.Opening.Tiles)], "the tile was blank at the reset")
	require.Len(t, out.Scenes, 1)
	assert.Equal(t, "de", out.Scenes[0].Change.Value)
}

func TestAReplayThatEndedBeforeAResetIsRefused(t *testing.T) {
	h := setup(t, "de")
	reset(t, h.resets, "season-1", start.Add(30*time.Minute))

	_, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start, Until: start.Add(10 * time.Minute)})

	require.ErrorIs(t, err, replay_usecase.ErrBeforeReset)
}

func TestAResetBeforeTheWindowChangesNothing(t *testing.T) {
	h := setup(t, "de")
	reset(t, h.resets, "season-1", start.Add(-time.Hour))

	out, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start})

	require.NoError(t, err)
	assert.Equal(t, start, out.Since)
}

func TestAReplayOpensOnTheMapAtSinceAndShowsWhatCameAfter(t *testing.T) {
	h := setup(t, "de")
	h.book.Append(ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start})
	h.book.Append(ledger.Taking{Tile: 1, Scope: "b", Country: "de", Previous: "fr", At: start.Add(time.Minute)})

	out, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start})
	require.NoError(t, err)

	assert.Equal(t, start, out.Since)
	assert.Equal(t, start.Add(time.Hour), out.Until, "an unset until is now")
	assert.Equal(t, [2]uint32{1, math.MaxUint32}, h.now.asked, "the whole map, from the first tile")
	assert.Equal(t, uint16(len(out.Opening.Codes)-1), binary.LittleEndian.Uint16(out.Opening.Tiles))
	assert.Equal(t, "it", out.Opening.Codes[len(out.Opening.Codes)-1])
	require.Len(t, out.Scenes, 2)
	assert.Equal(t, "fr", out.Scenes[0].Change.Value)
	assert.Equal(t, "de", out.Scenes[1].Change.Value)
}

func TestAReplayStopsAtUntil(t *testing.T) {
	h := setup(t, "de")
	h.book.Append(ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start})
	h.book.Append(ledger.Taking{Tile: 1, Scope: "b", Country: "de", Previous: "fr", At: start.Add(time.Minute)})

	out, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start, Until: start.Add(time.Second)})
	require.NoError(t, err)

	require.Len(t, out.Scenes, 1)
	assert.Equal(t, "fr", out.Scenes[0].Change.Value)
}

func TestAReplayNeedsASinceBeforeItsUntil(t *testing.T) {
	h := setup(t, "de")

	for _, in := range []replay_usecase.In{
		{},
		{Since: start, Until: start},
		{Since: start.Add(2 * time.Hour)},
	} {
		_, err := h.useCase.Execute(t.Context(), in)
		assert.ErrorIs(t, err, replay_usecase.ErrInvalidWindow)
	}
}

func TestAMapThatCannotBeReadFailsTheReplay(t *testing.T) {
	h := setup(t, "de")
	h.now.err = errors.New("no map")

	_, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start})

	require.ErrorIs(t, err, h.now.err)
}

func (h harness) arena(t *testing.T, id arenas.ID, openedAt time.Time, closedAt time.Time) {
	t.Helper()
	require.NoError(t, h.arenas.Open(t.Context(), arenas.Opening{ID: id, OpenedAt: openedAt, EndsAt: closedAt}))
	if !closedAt.IsZero() {
		require.NoError(t, h.arenas.Close(t.Context(), arenas.Closing{ID: id, ClosedAt: closedAt}))
	}
}

func ownersOf(batch clicks.DenseBatch) []string {
	owners := make([]string, len(batch.Tiles)/2)
	for i := range owners {
		owners[i] = batch.Codes[binary.LittleEndian.Uint16(batch.Tiles[i*2:])]
	}
	return owners
}

func TestAReplayAcrossAnArenaPlaysTheWorldsMapStraightThrough(t *testing.T) {
	h := setup(t, "fr", "de", "pl")
	opened, closed := start.Add(10*time.Minute+300*time.Millisecond), start.Add(40*time.Minute+700*time.Millisecond)
	h.arena(t, "battle", opened, closed)
	h.book.Append(ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start.Add(time.Minute)})
	h.book.Append(ledger.Taking{Tile: 2, Scope: "b", Country: "br", Previous: "", At: opened.Add(time.Minute)})
	h.book.Append(ledger.Taking{Tile: 1, Scope: "b", Country: "br", Previous: "", At: opened.Add(2 * time.Minute)})
	h.book.Append(ledger.Taking{Tile: 2, Scope: "c", Country: "de", Previous: "pl", At: closed.Truncate(time.Second)})

	out, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start})
	require.NoError(t, err)

	assert.Equal(t, []string{"it", "pl", "pl"}, ownersOf(out.Opening), "every tile opens on the world's flag")
	require.Len(t, out.Scenes, 2, "no scene of the battle")
	assert.Equal(t, "fr", out.Scenes[0].Change.Value)
	assert.Equal(t, "de", out.Scenes[1].Change.Value, "a world take in the second the arena closed")
	require.Len(t, out.Asides, 1)
	assert.Equal(t, arenas.ID("battle"), out.Asides[0].ID)
}

func TestAReplayWhileAnArenaPlaysRewindsTheWorldsMapKeptAside(t *testing.T) {
	h := setup(t, "br", "", "")
	h.now.kept = []string{"fr", "de", "pl"}
	opened := start.Add(30 * time.Minute)
	h.arena(t, "battle", opened, time.Time{})
	h.book.Append(ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start.Add(time.Minute)})
	h.book.Append(ledger.Taking{Tile: 1, Scope: "b", Country: "br", Previous: "", At: opened.Add(time.Minute)})

	out, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start})
	require.NoError(t, err)

	assert.Equal(t, []string{"it", "de", "pl"}, ownersOf(out.Opening), "rewound from the world's map, not the arena's")
	require.Len(t, out.Scenes, 1)
	assert.Equal(t, "fr", out.Scenes[0].Change.Value)
	require.Len(t, out.Asides, 1)
	assert.False(t, out.Asides[0].Closed, "the arena still plays")
}
