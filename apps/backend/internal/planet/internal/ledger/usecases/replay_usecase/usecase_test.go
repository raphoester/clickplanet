package replay_usecase_test

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/usecases/replay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/world_replay"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/resets"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var start = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

type tiles struct {
	// The world's on earth-1 when unset.
	playing clicks.Stage
	owners  []string
	kept    []string
	asked   [2]uint32
	err     error
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

func (t *tiles) Stage() clicks.Stage {
	if t.playing == (clicks.Stage{}) {
		return clicks.WorldStage
	}
	return t.playing
}

func (t *tiles) Kept(_ context.Context, kept clicks.Stage) (clicks.DenseBatch, error) {
	if kept != clicks.WorldStage {
		return clicks.DenseBatch{}, errors.New("only the world's map is kept aside, on the board of its era")
	}
	return batchOf(1, t.kept), t.err
}

type behind struct {
	entries []ledger.Entry
	asked   []ledger.Position
}

func (b *behind) Before(_ context.Context, position ledger.Position) (*ledger.Erasure, error) {
	b.asked = append(b.asked, position)
	erasure := ledger.NewErasure()
	for _, entry := range b.entries {
		if _, _, err := erasure.Read(entry); err != nil {
			return nil, fmt.Errorf("failed to read an event behind the head: %w", err)
		}
	}
	return erasure, nil
}

type harness struct {
	book    *inmemory_ledger_storage.Storage
	behind  *behind
	now     *tiles
	resets  *resets.MemoryStore
	arenas  *arenas.MemoryStore
	eras    *eras.MemoryStore
	useCase *replay_usecase.UseCase
}

func setup(t *testing.T, owners ...string) harness {
	t.Helper()
	h := harness{
		book: inmemory_ledger_storage.New(inmemory_ledger_storage.Config{},
			inmemory_ledger_storage.NewMemoryPersistence(), slog.New(slog.DiscardHandler)),
		behind: &behind{}, now: &tiles{owners: owners}, resets: resets.NewMemoryStore(), arenas: arenas.NewMemoryStore(),
		eras: eras.NewMemoryStore(),
	}
	h.useCase = replay_usecase.New(world_replay.NewKeptReplay(h.book, h.behind), h.resets, h.arenas, h.eras, h.now, cptime.NewFixedClock(start.Add(time.Hour)))
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
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start})
	reset(t, h.resets, "season-1", start.Add(time.Minute))
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "b", Country: "de", Previous: "", At: start.Add(2 * time.Minute)})

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

func TestAReplayOverASwitchOfTheWorldsBoardStartsAtItOnTheMapCarriedOver(t *testing.T) {
	h := setup(t, "de")
	earth2 := clicks.World.On(eras.Earth2.ID)
	switched := start.Add(time.Minute)
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start})
	require.NoError(t, h.eras.Start(t.Context(), earth2, switched))
	h.book.Append(earth2, ledger.Taking{Tile: 1, Scope: "b", Country: "de", Previous: "pl", At: start.Add(2 * time.Minute)})

	out, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start})
	require.NoError(t, err)

	assert.Equal(t, switched, out.Since)
	assert.Equal(t, eras.Earth2, out.Board)
	assert.Equal(t, []eras.Era{{Map: clicks.World, Board: eras.Earth2, StartedAt: switched}}, out.Eras)
	assert.Equal(t, "pl", out.Opening.Codes[binary.LittleEndian.Uint16(out.Opening.Tiles)],
		"earth-2's tile as the switch carried it over, rewound from now")
	require.Len(t, out.Scenes, 1, "earth-1's take before the switch is another board's")
	assert.Equal(t, "de", out.Scenes[0].Change.Value)
}

func TestAReplayThatEndedBeforeASwitchOfTheWorldsBoardIsRefused(t *testing.T) {
	h := setup(t, "de")
	require.NoError(t, h.eras.Start(t.Context(), clicks.World.On(eras.Earth2.ID), start.Add(30*time.Minute)))

	_, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start, Until: start.Add(10 * time.Minute)})

	require.ErrorIs(t, err, replay_usecase.ErrBeforeSwitch)
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
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start})
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "b", Country: "de", Previous: "fr", At: start.Add(time.Minute)})

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
	assert.Equal(t, eras.Earth1, out.Board, "the board the opening's tiles are on")
	assert.Equal(t, []eras.Era{{Map: clicks.World, Board: eras.Earth1, StartedAt: eras.Epoch}}, out.Eras)
}

func TestAReplayFilmsTheWorldsBoardAlone(t *testing.T) {
	h := setup(t, "de")
	h.book.Append(clicks.World.On(clicks.TestBoard+1),
		ledger.Taking{Tile: 1, Scope: "a", Country: "pl", Previous: "it", At: start.Add(time.Minute)})
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "b", Country: "de", Previous: "fr", At: start.Add(2 * time.Minute)})

	out, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start})
	require.NoError(t, err)

	assert.Equal(t, []string{"fr"}, ownersOf(out.Opening), "tile 1 of another board is another place")
	require.Len(t, out.Scenes, 1)
	assert.Equal(t, "de", out.Scenes[0].Change.Value)
}

func TestAReplayStopsAtUntil(t *testing.T) {
	h := setup(t, "de")
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start})
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "b", Country: "de", Previous: "fr", At: start.Add(time.Minute)})

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
	battle := clicks.MapID("battle").On(clicks.TestBoard)
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start.Add(time.Minute)})
	h.book.Append(battle, ledger.Taking{Tile: 2, Scope: "b", Country: "br", Previous: "", At: opened.Add(time.Minute)})
	h.book.Append(battle, ledger.Taking{Tile: 1, Scope: "b", Country: "br", Previous: "", At: opened.Add(2 * time.Minute)})
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 2, Scope: "c", Country: "de", Previous: "pl", At: closed.Truncate(time.Second)})

	out, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start})
	require.NoError(t, err)

	assert.Equal(t, []string{"it", "pl", "pl"}, ownersOf(out.Opening), "every tile opens on the world's flag")
	require.Len(t, out.Scenes, 2, "no scene of the battle")
	assert.Equal(t, "fr", out.Scenes[0].Change.Value)
	assert.Equal(t, "de", out.Scenes[1].Change.Value, "a world take, as its event names the world's map")
	require.Len(t, out.Asides, 1)
	assert.Equal(t, arenas.ID("battle"), out.Asides[0].ID)
}

func TestAReplayWhileAnArenaPlaysRewindsTheWorldsMapKeptAside(t *testing.T) {
	h := setup(t, "br", "", "")
	h.now.kept = []string{"fr", "de", "pl"}
	opened := start.Add(30 * time.Minute)
	h.arena(t, "battle", opened, time.Time{})
	h.now.playing = clicks.MapID("battle").On(clicks.TestBoard)
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "a", Country: "fr", Previous: "it", At: start.Add(time.Minute)})
	h.book.Append(h.now.playing,
		ledger.Taking{Tile: 1, Scope: "b", Country: "br", Previous: "", At: opened.Add(time.Minute)})

	out, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start})
	require.NoError(t, err)

	assert.Equal(t, []string{"it", "de", "pl"}, ownersOf(out.Opening), "rewound from the world's map, not the arena's")
	require.Len(t, out.Scenes, 1)
	assert.Equal(t, "fr", out.Scenes[0].Change.Value)
	require.Len(t, out.Asides, 1)
	assert.False(t, out.Asides[0].Closed, "the arena still plays")
}

func TestAReplayOpensARevertedDrawingPaintedOverOnWhatWasBeneathIt(t *testing.T) {
	h := setup(t, "ie", "fr")
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "v", Country: "cn", Previous: "ie", At: start})
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 2, Scope: "v", Country: "cn", Previous: "ie", At: start.Add(2 * time.Minute)})
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "b", Country: "ie", Previous: "cn", At: start.Add(3 * time.Minute)})
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 2, Scope: "c", Country: "fr", Previous: "cn", At: start.Add(4 * time.Minute)})
	h.book.Forget(ledger.Caller{Scope: "v"}, h.book.Replay(func(clicks.Stage, ledger.Event) {}))

	out, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start.Add(time.Minute)})
	require.NoError(t, err)

	assert.Equal(t, []string{"ie", "ie"}, ownersOf(out.Opening), "the drawing never shows")
	require.Len(t, out.Scenes, 1, "Ireland painting over it changed nothing")
	assert.Equal(t, clicks.TileUpdate{Tile: 2, Value: "fr", Previous: "ie", Clicked: true}, out.Scenes[0].Change.TileUpdate)
}

func TestAReplayOpensADrawingForgottenBeforeTheLedgersHeadAndPaintedOverOnWhatWasBeneathIt(t *testing.T) {
	h := setup(t, "ie", "fr")
	painted := start.Add(-80 * time.Hour)
	paint := []ledger.Taking{
		{Tile: 1, Scope: "v", Country: "cn", Previous: "ie", At: painted},
		{Tile: 2, Scope: "v", Country: "cn", Previous: "ie", At: painted},
	}
	for _, taking := range paint {
		h.book.Append(clicks.WorldStage, taking)
		entry, err := taking.Entry()
		require.NoError(t, err)
		entry.Stage, entry.Forgotten = clicks.WorldStage, true
		h.behind.entries = append(h.behind.entries, entry)
	}
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 1, Scope: "b", Country: "ie", Previous: "cn", At: start.Add(3 * time.Minute)})
	h.book.Append(clicks.WorldStage, ledger.Taking{Tile: 2, Scope: "c", Country: "fr", Previous: "cn", At: start.Add(4 * time.Minute)})
	h.book.ForgetBefore(start.Add(-72 * time.Hour))

	out, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start})
	require.NoError(t, err)

	assert.Equal(t, []ledger.Position{2}, h.behind.asked, "the erasure before the head, which memory no longer keeps")
	assert.Equal(t, []string{"ie", "ie"}, ownersOf(out.Opening), "the drawing never shows")
	require.Len(t, out.Scenes, 1, "Ireland painting over it changed nothing")
	assert.Equal(t, clicks.TileUpdate{Tile: 2, Value: "fr", Previous: "ie", Clicked: true}, out.Scenes[0].Change.TileUpdate)
}

func TestAReplayFailsWhenTheErasureBeforeTheHeadCannotBeRead(t *testing.T) {
	h := setup(t, "de")
	h.behind.entries = []ledger.Entry{{Stage: clicks.WorldStage, Kind: "spread", Payload: []byte(`{"taken":"not a list"}`)}}

	_, err := h.useCase.Execute(t.Context(), replay_usecase.In{Since: start})

	require.Error(t, err)
}
