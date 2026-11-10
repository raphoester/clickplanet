package open_arena_usecase_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/arenas/usecases/open_arena_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
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
	outbox  *cpatomic.FakeOutbox
	inPlay  *arenas.InPlay
	clock   *cptime.FixedClock
	useCase *open_arena_usecase.UseCase
}

func newHarness(t *testing.T) harness {
	t.Helper()
	clock := cptime.NewFixedClock(starts)
	stored := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{1: "fr", 2: "fr", 3: "de"})
	tiles := inmemory_tile_storage.New(clicks.BordersOf(100), inmemory_tile_storage.Config{}, stored,
		cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{}, clock,
		slog.New(slog.DiscardHandler))
	require.NoError(t, tiles.Load(t.Context(), clicks.World))
	kept, outbox := arenas.NewMemoryStore(), &cpatomic.FakeOutbox{}
	inPlay := arenas.NewInPlay(func() arenas.Rules { return arenas.Rules{Freeze: 8 * time.Second} }, clock)
	return harness{
		tiles: tiles, stored: stored, kept: kept, outbox: outbox, inPlay: inPlay, clock: clock,
		useCase: open_arena_usecase.New(tiles, kept, outbox, inPlay, clock),
	}
}

func (h harness) owner(tile uint32) string {
	owner, _ := h.tiles.Owner(tile)
	return owner
}

func TestAnArenaOpensOnABlankMapAndLeavesTheWorldsAsItIs(t *testing.T) {
	h := newHarness(t)
	require.NoError(t, h.tiles.Set(t.Context(), 4, "it"))

	opened, err := h.useCase.Execute(t.Context(), open_arena_usecase.Command{Arena: "battle", EndsAt: ends})

	require.NoError(t, err)
	assert.True(t, opened)
	for tile := range uint32(5) {
		assert.Empty(t, h.owner(tile), "tile %d", tile)
	}
	assert.Equal(t, map[uint32]string{1: "fr", 2: "fr", 3: "de", 4: "it"}, h.stored.Stored(), "the tile not flushed yet too")
	assert.Empty(t, h.stored.StoredOn("battle"))
	assert.True(t, bool(h.inPlay.Freeze()), "the map takes no play right after it went blank")
	h.clock.Advance(8 * time.Second)
	assert.False(t, bool(h.inPlay.Freeze()))
	playing, err := h.kept.Playing(t.Context())
	require.NoError(t, err)
	assert.Equal(t, clicks.MapID("battle"), playing.Played())
	require.Len(t, h.outbox.Recorded(), 1)
	assert.True(t, proto.Equal(&planetv1.ArenaOpened{ArenaId: "battle", OpenedAt: timestamppb.New(starts)},
		h.outbox.Recorded()[0]))
}

func TestAnArenaToldTwiceOpensOnce(t *testing.T) {
	h := newHarness(t)
	command := open_arena_usecase.Command{Arena: "battle", EndsAt: ends}
	_, err := h.useCase.Execute(t.Context(), command)
	require.NoError(t, err)
	require.NoError(t, h.tiles.Set(t.Context(), 7, "pt"))

	opened, err := h.useCase.Execute(t.Context(), command)

	require.NoError(t, err)
	assert.False(t, opened)
	assert.Equal(t, "pt", h.owner(7), "the battle's map stays")
	assert.Len(t, h.outbox.Recorded(), 1, "told once")
}

func TestAnArenaClosedEndedOrBesideAnotherOpensNothing(t *testing.T) {
	for name, setUp := range map[string]func(h harness) open_arena_usecase.Command{
		"closed": func(h harness) open_arena_usecase.Command {
			require.NoError(t, h.kept.Close(t.Context(), arenas.Closing{ID: "battle", ClosedAt: starts}))
			return open_arena_usecase.Command{Arena: "battle", EndsAt: ends}
		},
		"ended": func(harness) open_arena_usecase.Command {
			return open_arena_usecase.Command{Arena: "battle", EndsAt: starts}
		},
		"another open": func(h harness) open_arena_usecase.Command {
			require.NoError(t, h.kept.Open(t.Context(), arenas.Opening{ID: "other", EndsAt: ends, OpenedAt: starts}))
			return open_arena_usecase.Command{Arena: "battle", EndsAt: ends}
		},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)

			opened, err := h.useCase.Execute(t.Context(), setUp(h))

			require.ErrorIs(t, err, arenas.ErrRefused)
			assert.False(t, opened)
			assert.Equal(t, "fr", h.owner(1), "the map stays")
			assert.False(t, bool(h.inPlay.Freeze()))
			assert.Empty(t, h.outbox.Recorded())
		})
	}
}
