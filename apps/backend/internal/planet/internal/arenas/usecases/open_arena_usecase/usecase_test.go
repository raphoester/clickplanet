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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/seeds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var capitals = clicks.NewCapitals(map[string]uint32{"fr": 50, "de": 60})

var (
	starts = time.Date(2026, 11, 10, 21, 0, 3, 0, time.UTC)
	ends   = time.Date(2026, 11, 10, 23, 0, 0, 0, time.UTC)
)

type harness struct {
	tiles   *inmemory_tile_storage.Storage
	stored  *inmemory_tile_storage.MemoryPersistence
	kept    *arenas.MemoryStore
	inPlay  *arenas.InPlay
	clock   *cptime.FixedClock
	useCase *open_arena_usecase.UseCase
}

func newHarness(t *testing.T) harness {
	t.Helper()
	clock := cptime.NewFixedClock(starts)
	stored := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{1: "fr", 2: "fr", 3: "de"})
	tiles := inmemory_tile_storage.New(clicks.BordersOf(100), clicks.CapitalsPlayed(capitals), inmemory_tile_storage.Config{}, stored,
		cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{}, clock,
		slog.New(slog.DiscardHandler))
	require.NoError(t, tiles.Load(t.Context(), clicks.World))
	kept := arenas.NewMemoryStore()
	inPlay := arenas.NewInPlay(clock)
	sets := cprules.NewMemoryStore(parts.All(), rules.Seeded(), rules.Named(battleSet, func(r *planetv1.Rules) {
		r.Arena = &planetv1.ArenaRules{On: new(true), Freeze: durationpb.New(6 * time.Second)}
		r.Seed = &planetv1.Seed{Pattern: &planetv1.Seed_Capitals{Capitals: &planetv1.CapitalSeed{Shields: new(uint32(40))}}}
	}))
	setups := arenas.NewSetups(sets, seeds.NewCatalog(seeds.Ground{Capitals: capitals}))
	return harness{
		tiles: tiles, stored: stored, kept: kept, inPlay: inPlay, clock: clock,
		useCase: open_arena_usecase.New(tiles, setups, kept, inPlay, clock),
	}
}

const battleSet cprules.SetName = "siege"

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
