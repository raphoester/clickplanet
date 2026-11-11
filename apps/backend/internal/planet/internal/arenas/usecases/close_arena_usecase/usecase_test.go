package close_arena_usecase_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/close_arena_usecase"
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
	opening *open_arena_usecase.UseCase
	useCase *close_arena_usecase.UseCase
}

func newHarness(t *testing.T) harness {
	t.Helper()
	clock := cptime.NewFixedClock(starts)
	stored := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{1: "fr", 2: "fr", 3: "de"})
	stored.Shield(1, 4)
	tiles := inmemory_tile_storage.New(clicks.BordersOf(100), clicks.NoCapitals(), inmemory_tile_storage.Config{}, stored,
		cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{}, clock,
		slog.New(slog.DiscardHandler))
	require.NoError(t, tiles.Load(t.Context(), clicks.World))
	kept := arenas.NewMemoryStore()
	inPlay := arenas.NewInPlay(clock)
	setups := arenas.NewSetups(cprules.NewMemoryStore(parts.All(), rules.Seeded()),
		seeds.NewCatalog(seeds.Ground{Capitals: clicks.NewCapitals(map[string]uint32{})}))
	return harness{
		tiles: tiles, stored: stored, kept: kept, inPlay: inPlay, clock: clock,
		opening: open_arena_usecase.New(tiles, setups, kept, inPlay, clock),
		useCase: close_arena_usecase.New(tiles, kept, inPlay, clock),
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
