package close_arena_usecase_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/close_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/open_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/seeds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	starts = time.Date(2026, 11, 10, 21, 0, 3, 0, time.UTC)
	ends   = time.Date(2026, 11, 10, 23, 0, 0, 0, time.UTC)
)

// The world's ground, and a board of its own an arena may play: 30 tiles.
var (
	world = clicks.NamedGround(eras.Earth1.Name, clicks.BordersOf(100), clicks.NewCapitals(map[string]uint32{}))
	small = clicks.NamedGround(eras.Fixture.Name, clicks.BordersOf(30), clicks.NewCapitals(map[string]uint32{}))
)

// A set whose arena plays a board of its own.
const boardSet cprules.SetName = "siege-board"

type harness struct {
	tiles   *inmemory_tile_storage.Storage
	played  *clicks.Played
	stored  *inmemory_tile_storage.MemoryPersistence
	kept    *arenas.MemoryStore
	eras    *eras.MemoryStore
	inPlay  *arenas.InPlay
	clock   *cptime.FixedClock
	opening *open_arena_usecase.UseCase
	useCase *close_arena_usecase.UseCase
}

func newHarness(t *testing.T) harness {
	t.Helper()
	clock := cptime.NewFixedClock(starts)
	stored := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{1: "fr", 2: "fr", 3: "de"})
	stored.Shield(1, 4)
	grounds := eras.NewGrounds(eras.CarryingFixture, func(name clicks.BoardName) (*clicks.Ground, error) {
		if name == small.Name() {
			return small, nil
		}
		return world, nil
	})
	played := clicks.NewPlayed()
	tiles := inmemory_tile_storage.New(grounds, played, clicks.NoCapitals(), inmemory_tile_storage.Config{}, stored,
		cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{}, clock,
		slog.New(slog.DiscardHandler))
	require.NoError(t, tiles.Load(t.Context(), clicks.WorldStage))
	kept := arenas.NewMemoryStore()
	keptEras := eras.NewMemoryStore()
	keptEras.Keep(eras.Fixture)
	inPlay := arenas.NewInPlay(clock)
	setups := arenas.NewSetups(cprules.NewMemoryStore(parts.All(), rules.Seeded(), rules.Named(boardSet, func(r *planetv1.Rules) {
		r.Arena = &planetv1.ArenaRules{On: new(true), Board: new(string(eras.Fixture.Name))}
	})), seeds.Catalog{}, grounds)
	return harness{
		tiles: tiles, played: played, stored: stored, kept: kept, eras: keptEras, inPlay: inPlay, clock: clock,
		opening: open_arena_usecase.New(tiles, setups, kept, keptEras, inPlay, clock),
		useCase: close_arena_usecase.New(tiles, eras.NewMaps(keptEras, eras.CarryingFixture), kept, inPlay, clock),
	}
}

func (h harness) owner(tile uint32) string {
	owner, _ := h.tiles.Owner(tile)
	return owner
}

func (h harness) closedAt(t *testing.T) time.Time {
	t.Helper()
	stretches, err := h.kept.Stretches(t.Context())
	require.NoError(t, err)
	require.Len(t, stretches, 1)
	return stretches[0].ClosedAt
}

func TestAnArenaClosedBringsTheMapFromBeforeItBackOnce(t *testing.T) {
	h := newHarness(t)
	_, err := h.opening.Execute(t.Context(), open_arena_usecase.Command{Arena: "battle", EndsAt: ends, Set: cprules.Default})
	require.NoError(t, err)
	require.NoError(t, h.tiles.Set(t.Context(), 1, "it"))
	require.NoError(t, h.tiles.Set(t.Context(), 9, "it"))
	h.clock.Advance(ends.Add(3 * time.Second).Sub(starts))
	require.True(t, bool(h.inPlay.Freeze()), "the arena takes no play until the map is back")

	closed, err := h.useCase.Execute(t.Context(), "battle")

	require.NoError(t, err)
	assert.Equal(t, close_arena_usecase.Restored, closed)
	assert.Equal(t, map[uint32]string{1: "fr", 2: "fr", 3: "de"}, h.stored.Stored())
	assert.Equal(t, map[uint32]int{1: 4}, h.stored.StoredShields())
	assert.Equal(t, "fr", h.owner(1))
	assert.Empty(t, h.owner(9))
	assert.Equal(t, 4, h.tiles.Shields(1))
	assert.False(t, bool(h.inPlay.Freeze()))
	closing := h.clock.Now()
	assert.Equal(t, closing, h.closedAt(t))

	require.NoError(t, h.tiles.Set(t.Context(), 9, "pt"))
	h.clock.Advance(time.Minute)
	closed, err = h.useCase.Execute(t.Context(), "battle")
	require.NoError(t, err)
	assert.Equal(t, close_arena_usecase.Already, closed)
	assert.Equal(t, "pt", h.owner(9), "a close asked again changes nothing")
	assert.Equal(t, closing, h.closedAt(t), "kept as it closed")
}

func TestAnArenaNeverOpenedClosesWithTheMapLeftAsItIs(t *testing.T) {
	h := newHarness(t)
	require.NoError(t, h.tiles.Set(t.Context(), 9, "it"))

	closed, err := h.useCase.Execute(t.Context(), "battle")

	require.NoError(t, err)
	assert.Equal(t, close_arena_usecase.Unopened, closed)
	assert.Equal(t, "it", h.owner(9))
	assert.Equal(t, "fr", h.owner(1))
	_, err = h.opening.Execute(t.Context(), open_arena_usecase.Command{Arena: "battle", EndsAt: ends, Set: cprules.Default})
	require.ErrorIs(t, err, arenas.ErrRefused, "an arena closed never opens")
}

func TestAnArenaThatCannotCloseLeavesTheMap(t *testing.T) {
	h := newHarness(t)
	_, err := h.opening.Execute(t.Context(), open_arena_usecase.Command{Arena: "battle", EndsAt: ends, Set: cprules.Default})
	require.NoError(t, err)
	require.NoError(t, h.tiles.Set(t.Context(), 9, "it"))
	down := errors.New("postgres is down")
	h.kept.Failing = down

	_, err = h.useCase.Execute(t.Context(), "battle")

	require.ErrorIs(t, err, down)
	assert.Equal(t, "it", h.owner(9))
	assert.True(t, bool(h.inPlay.Freeze()), "still open")
}

func TestAnArenaOnABoardOfItsOwnClosesOnTheWorldsEraAgain(t *testing.T) {
	h := newHarness(t)
	_, err := h.opening.Execute(t.Context(), open_arena_usecase.Command{Arena: "battle", EndsAt: ends, Set: boardSet})
	require.NoError(t, err)
	require.Equal(t, clicks.MapID("battle").On(eras.Fixture.ID), h.tiles.Stage())
	require.NoError(t, h.tiles.Set(t.Context(), 30, "it"))
	h.clock.Advance(ends.Add(3 * time.Second).Sub(starts))

	closed, err := h.useCase.Execute(t.Context(), "battle")

	require.NoError(t, err)
	assert.Equal(t, close_arena_usecase.Restored, closed)
	assert.Equal(t, clicks.WorldStage, h.tiles.Stage(), "the world plays the board of its era again")
	assert.Equal(t, eras.Earth1.Name, h.played.Name(), "every rule reads the world's board again")
	assert.Equal(t, uint32(100), h.played.Tiles())
	assert.Equal(t, "fr", h.owner(1))
	assert.Equal(t, 4, h.tiles.Shields(1))
	assert.Empty(t, h.owner(30), "the arena's take stayed on its board")
	world, err := h.eras.Latest(t.Context(), clicks.World)
	require.NoError(t, err)
	assert.Equal(t, eras.Era{Map: clicks.World, Board: eras.Earth1, StartedAt: eras.Epoch}, world, "the world's era resumes")
}
