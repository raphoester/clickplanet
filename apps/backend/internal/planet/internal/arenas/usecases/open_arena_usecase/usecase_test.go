package open_arena_usecase_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
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

var capitals = clicks.NewCapitals(map[string]uint32{"fr": 50, "de": 60})

// The world's ground, and a board of its own an arena may play: 30 tiles, with capitals of their own.
var (
	world = clicks.NamedGround(eras.Earth1.Name, clicks.BordersOf(100), capitals)
	small = clicks.NamedGround(eras.Fixture.Name, clicks.BordersOf(30), clicks.NewCapitals(map[string]uint32{"fr": 7, "jp": 9}))
)

// groundsOf are the grounds of the boards planet keeps: by name, and by row.
func groundsOf(kept eras.Boards) eras.Grounds {
	return eras.NewGrounds(kept, func(name clicks.BoardName) (*clicks.Ground, error) {
		if name == small.Name() {
			return small, nil
		}
		return world, nil
	})
}

var (
	starts = time.Date(2026, 11, 10, 21, 0, 3, 0, time.UTC)
	ends   = time.Date(2026, 11, 10, 23, 0, 0, 0, time.UTC)
)

type harness struct {
	tiles   *inmemory_tile_storage.Storage
	played  *clicks.Played
	stored  *inmemory_tile_storage.MemoryPersistence
	kept    *arenas.MemoryStore
	eras    *eras.MemoryStore
	inPlay  *arenas.InPlay
	clock   *cptime.FixedClock
	useCase *open_arena_usecase.UseCase
}

func newHarness(t *testing.T) harness {
	t.Helper()
	return newHarnessCarrying(t, eras.CarryingFixture)
}

// newHarnessCarrying is a harness whose process carries the boards: earth-1, which the world plays, and the boards
// beside it.
func newHarnessCarrying(t *testing.T, carried eras.Boards) harness {
	t.Helper()
	clock := cptime.NewFixedClock(starts)
	stored := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{1: "fr", 2: "fr", 3: "de"})
	grounds := groundsOf(carried)
	played := clicks.NewPlayed()
	tiles := inmemory_tile_storage.New(grounds, played, clicks.NewCapitalPlay(played.Capitals, func() clicks.CapitalRules {
		return clicks.CapitalRules{KnockOut: true, FixedShields: true}
	}), inmemory_tile_storage.Config{}, stored, cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{},
		clock, slog.New(slog.DiscardHandler))
	require.NoError(t, tiles.Load(t.Context(), clicks.WorldStage))
	kept := arenas.NewMemoryStore()
	keptEras := eras.NewMemoryStore()
	keptEras.Keep(eras.Fixture)
	inPlay := arenas.NewInPlay(clock)
	sets := cprules.NewMemoryStore(parts.All(), rules.Seeded(), rules.Named(battleSet, func(r *planetv1.Rules) {
		r.Arena = &planetv1.ArenaRules{On: new(true), Freeze: durationpb.New(6 * time.Second)}
		r.Seed = &planetv1.Seed{Pattern: &planetv1.Seed_Capitals{Capitals: &planetv1.CapitalSeed{Shields: new(uint32(40))}}}
	}), rules.Named(boardSet, func(r *planetv1.Rules) {
		r.Arena = &planetv1.ArenaRules{On: new(true), Freeze: durationpb.New(6 * time.Second), Board: new(string(eras.Fixture.Name))}
		r.Seed = &planetv1.Seed{Pattern: &planetv1.Seed_Capitals{Capitals: &planetv1.CapitalSeed{Shields: new(uint32(40))}}}
	}))
	setups := arenas.NewSetups(sets, seeds.Catalog{}, grounds)
	return harness{
		tiles: tiles, played: played, stored: stored, kept: kept, eras: keptEras, inPlay: inPlay, clock: clock,
		useCase: open_arena_usecase.New(tiles, setups, kept, keptEras, inPlay, clock),
	}
}

const (
	battleSet cprules.SetName = "siege"
	// A set whose arena plays a board of its own.
	boardSet cprules.SetName = "siege-board"
)

func command(endsAt time.Time) open_arena_usecase.Command {
	return open_arena_usecase.Command{Arena: "battle", EndsAt: endsAt, Set: battleSet}
}

func (h harness) owner(tile uint32) string {
	owner, _ := h.tiles.Owner(tile)
	return owner
}

func TestAnArenaOpensOnABlankMapAndLeavesTheWorldsAsItIs(t *testing.T) {
	h := newHarness(t)
	require.NoError(t, h.tiles.Set(t.Context(), 4, "it"))

	opened, err := h.useCase.Execute(t.Context(), command(ends))

	require.NoError(t, err)
	assert.True(t, opened)
	for tile := range uint32(5) {
		assert.Empty(t, h.owner(tile), "tile %d", tile)
	}
	assert.Equal(t, map[uint32]string{1: "fr", 2: "fr", 3: "de", 4: "it"}, h.stored.Stored(), "the tile not flushed yet too")
	assert.Equal(t, map[uint32]string{50: "fr", 60: "de"}, h.stored.StoredOn("battle"), "the capitals alone")
	assert.True(t, bool(h.inPlay.Freeze()), "the map takes no play right after it went blank")
	h.clock.Advance(6 * time.Second)
	assert.False(t, bool(h.inPlay.Freeze()))
	playing, err := h.kept.Playing(t.Context())
	require.NoError(t, err)
	assert.Equal(t, clicks.MapID("battle"), playing.Played())
	assert.Equal(t, 6*time.Second, playing.Opening.Freeze, "the battle's set's freeze, kept for a boot")
	assert.Equal(t, starts, playing.Opening.OpenedAt)
	era, err := h.eras.Latest(t.Context(), "battle")
	require.NoError(t, err)
	assert.Equal(t, eras.Era{Map: "battle", Board: eras.Earth1, StartedAt: starts}, era,
		"the arena's era, on the board the world played, from its open")
	assert.Equal(t, clicks.MapID("battle").On(eras.Earth1.ID), h.tiles.Stage())
	assert.Equal(t, eras.Earth1.ID, h.stored.BoardOn("battle"), "its rows name the board")
}

func TestAnArenaToldTwiceOpensOnce(t *testing.T) {
	h := newHarness(t)
	command := command(ends)
	_, err := h.useCase.Execute(t.Context(), command)
	require.NoError(t, err)
	require.NoError(t, h.tiles.Set(t.Context(), 7, "pt"))

	opened, err := h.useCase.Execute(t.Context(), command)

	require.NoError(t, err)
	assert.False(t, opened)
	assert.Equal(t, "pt", h.owner(7), "the battle's map stays")
}

func TestAnArenaClosedEndedOrBesideAnotherOpensNothing(t *testing.T) {
	for name, setUp := range map[string]func(h harness) open_arena_usecase.Command{
		"closed": func(h harness) open_arena_usecase.Command {
			require.NoError(t, h.kept.Close(t.Context(), arenas.Closing{ID: "battle", ClosedAt: starts}))
			return command(ends)
		},
		"ended": func(harness) open_arena_usecase.Command {
			return command(starts)
		},
		"another open": func(h harness) open_arena_usecase.Command {
			require.NoError(t, h.kept.Open(t.Context(), arenas.Opening{ID: "other", EndsAt: ends, OpenedAt: starts}))
			return command(ends)
		},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)

			opened, err := h.useCase.Execute(t.Context(), setUp(h))

			require.ErrorIs(t, err, arenas.ErrRefused)
			assert.False(t, opened)
			assert.Equal(t, "fr", h.owner(1), "the map stays")
			assert.False(t, bool(h.inPlay.Freeze()))
			_, err = h.eras.Latest(t.Context(), "battle")
			assert.ErrorIs(t, err, eras.ErrNoEra, "no era for a map that never played")
		})
	}
}

func TestAnArenaOpensSeededAsTheBattlesSetSays(t *testing.T) {
	h := newHarness(t)

	opened, err := h.useCase.Execute(t.Context(), command(ends))

	require.NoError(t, err)
	assert.True(t, opened)
	assert.Equal(t, "fr", h.owner(50))
	assert.Equal(t, "de", h.owner(60))
	assert.Equal(t, 40, h.tiles.Shields(50))
	assert.Equal(t, map[uint32]string{50: "fr", 60: "de"}, h.stored.StoredOn("battle"), "kept with the arena, before a flush")
	assert.Equal(t, map[uint32]int{50: 40, 60: 40}, h.stored.StoredShieldsOn("battle"))
	assert.Equal(t, map[uint32]string{1: "fr", 2: "fr", 3: "de"}, h.stored.Stored(), "the world's map has no capital")
}

func TestAnArenaOfASetPlanetKeepsNoneOfIsRefused(t *testing.T) {
	h := newHarness(t)

	opened, err := h.useCase.Execute(t.Context(), open_arena_usecase.Command{Arena: "battle", EndsAt: ends, Set: "gone"})

	require.ErrorIs(t, err, arenas.ErrRefused)
	assert.False(t, opened)
	assert.Equal(t, "fr", h.owner(1), "the map stays")
}

func TestAnArenaOpensOnTheBoardItsSetNames(t *testing.T) {
	h := newHarness(t)

	opened, err := h.useCase.Execute(t.Context(),
		open_arena_usecase.Command{Arena: "battle", EndsAt: ends, Set: boardSet})

	require.NoError(t, err)
	assert.True(t, opened)
	era, err := h.eras.Latest(t.Context(), "battle")
	require.NoError(t, err)
	assert.Equal(t, eras.Era{Map: "battle", Board: eras.Fixture, StartedAt: starts}, era, "the arena's era, on its set's board")
	assert.Equal(t, clicks.MapID("battle").On(eras.Fixture.ID), h.tiles.Stage())
	assert.Equal(t, eras.Fixture.ID, h.stored.BoardOn("battle"), "its rows name its board")
	assert.Equal(t, eras.Fixture.Name, h.played.Name(), "every rule reads the arena's board")
	assert.Equal(t, uint32(30), h.played.Tiles())
	assert.Equal(t, map[uint32]string{7: "fr", 9: "jp"}, h.stored.StoredOn("battle"), "seeded with its board's capitals")
	assert.Equal(t, "jp", h.owner(9))
	assert.Empty(t, h.owner(31), "no tile past its board")
	world, err := h.eras.Latest(t.Context(), clicks.World)
	require.NoError(t, err)
	assert.Equal(t, eras.Earth1, world.Board, "the world's era stays")
}

func TestAnArenaOfABoardPlanetKeepsNoRowOfIsRefused(t *testing.T) {
	h := newHarnessCarrying(t, eras.CarryingEarth1)

	opened, err := h.useCase.Execute(t.Context(),
		open_arena_usecase.Command{Arena: "battle", EndsAt: ends, Set: boardSet})

	require.ErrorIs(t, err, arenas.ErrRefused)
	assert.False(t, opened)
	assert.Equal(t, "fr", h.owner(1), "the map stays")
	assert.Equal(t, eras.Earth1.Name, h.played.Name())
}
