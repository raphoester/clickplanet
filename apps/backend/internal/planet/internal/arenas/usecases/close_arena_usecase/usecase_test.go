package close_arena_usecase_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/close_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/open_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/seeds"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	starts = time.Date(2026, 11, 10, 21, 0, 3, 0, time.UTC)
	ends   = time.Date(2026, 11, 10, 23, 0, 0, 0, time.UTC)
)

// The storage's transaction, as postgres runs it: the outbox records in it.
type inTx struct {
	*inmemory_tile_storage.MemoryPersistence
}

func (p inTx) Wipe(ctx context.Context, played clicks.MapID, keep func(context.Context) (bool, error)) (bool, error) {
	var wiped bool
	err := cpatomic.FakeTransactioner{}.InTx(ctx, func(ctx context.Context) error {
		var err error
		wiped, err = p.MemoryPersistence.Wipe(ctx, played, keep)
		return err //nolint:wrapcheck // the persistence names it.
	})
	return wiped, err
}

func (p inTx) Open(
	ctx context.Context, played clicks.MapID, tiles []inmemory_tile_storage.Tile, keep func(context.Context) (bool, error),
) (bool, error) {
	var opened bool
	err := cpatomic.FakeTransactioner{}.InTx(ctx, func(ctx context.Context) error {
		var err error
		opened, err = p.MemoryPersistence.Open(ctx, played, tiles, keep)
		return err //nolint:wrapcheck // the persistence names it.
	})
	return opened, err
}

type harness struct {
	tiles   *inmemory_tile_storage.Storage
	stored  *inmemory_tile_storage.MemoryPersistence
	kept    *arenas.MemoryStore
	outbox  *cpatomic.FakeOutbox
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
	tiles := inmemory_tile_storage.New(clicks.BordersOf(100), clicks.NoCapitals(), inmemory_tile_storage.Config{}, inTx{stored},
		cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{}, clock,
		slog.New(slog.DiscardHandler))
	require.NoError(t, tiles.Load(t.Context(), clicks.World))
	kept, outbox := arenas.NewMemoryStore(), &cpatomic.FakeOutbox{}
	inPlay := arenas.NewInPlay(clock)
	setups := arenas.NewSetups(rules.NewMemoryStore(parts.All(), rules.Seeded()),
		seeds.NewCatalog(seeds.Ground{Capitals: clicks.NewCapitals(map[string]uint32{})}))
	return harness{
		tiles: tiles, stored: stored, kept: kept, outbox: outbox, inPlay: inPlay, clock: clock,
		opening: open_arena_usecase.New(tiles, setups, kept, &cpatomic.FakeOutbox{}, inPlay, clock),
		useCase: close_arena_usecase.New(tiles, kept, outbox, inPlay, clock),
	}
}

func (h harness) owner(tile uint32) string {
	owner, _ := h.tiles.Owner(tile)
	return owner
}

func closedAt(at time.Time) proto.Message {
	return &planetv1.ArenaClosed{ArenaId: "battle", ClosedAt: timestamppb.New(at)}
}

func TestAnArenaClosedBringsTheMapFromBeforeItBackAndIsToldAgainAtEachClose(t *testing.T) {
	h := newHarness(t)
	_, err := h.opening.Execute(t.Context(), open_arena_usecase.Command{Arena: "battle", EndsAt: ends, Set: rules.Default})
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
	require.Len(t, h.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(closedAt(closing), h.outbox.Recorded()[0]))

	require.NoError(t, h.tiles.Set(t.Context(), 9, "pt"))
	h.clock.Advance(time.Minute)
	closed, err = h.useCase.Execute(t.Context(), "battle")
	require.NoError(t, err)
	assert.Equal(t, close_arena_usecase.Already, closed)
	assert.Equal(t, "pt", h.owner(9), "a close told again changes nothing")
	require.Len(t, h.outbox.Recorded(), 2)
	assert.True(t, proto.Equal(closedAt(closing), h.outbox.Recorded()[1]), "told again as it closed")
}

func TestAnArenaNeverOpenedClosesAndIsToldWithTheMapLeftAsItIs(t *testing.T) {
	h := newHarness(t)
	require.NoError(t, h.tiles.Set(t.Context(), 9, "it"))

	closed, err := h.useCase.Execute(t.Context(), "battle")

	require.NoError(t, err)
	assert.Equal(t, close_arena_usecase.Unopened, closed)
	assert.Equal(t, "it", h.owner(9))
	assert.Equal(t, "fr", h.owner(1))
	require.Len(t, h.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(closedAt(starts), h.outbox.Recorded()[0]))
	_, err = h.opening.Execute(t.Context(), open_arena_usecase.Command{Arena: "battle", EndsAt: ends, Set: rules.Default})
	require.ErrorIs(t, err, arenas.ErrRefused, "an arena closed never opens")
}

func TestAnArenaThatCannotCloseLeavesTheMapAndTellsNothing(t *testing.T) {
	h := newHarness(t)
	_, err := h.opening.Execute(t.Context(), open_arena_usecase.Command{Arena: "battle", EndsAt: ends, Set: rules.Default})
	require.NoError(t, err)
	require.NoError(t, h.tiles.Set(t.Context(), 9, "it"))
	down := errors.New("postgres is down")
	h.kept.Failing = down

	_, err = h.useCase.Execute(t.Context(), "battle")

	require.ErrorIs(t, err, down)
	assert.Equal(t, "it", h.owner(9))
	assert.Empty(t, h.outbox.Recorded())
	assert.True(t, bool(h.inPlay.Freeze()), "still open")
}
