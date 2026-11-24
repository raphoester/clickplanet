package switch_board_usecase_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_board"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/inmemory_tile_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/eras/usecases/switch_board_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger/inmemory_ledger_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var now = time.Date(2026, 11, 22, 4, 0, 0, 0, time.UTC)

// earth-1 here is 3 tiles and earth-2 is 8: old tile 1 has children 1 to 4, 2 has 5 and 6, 3 has 7; 8 is a new island.
var parents = []uint32{1, 1, 1, 1, 2, 2, 3, 0}

type carryovers map[[2]clicks.BoardName]clicks.Carryover

func (c carryovers) Carryover(from, to clicks.BoardName) (clicks.Carryover, error) {
	carry, ok := c[[2]clicks.BoardName{from, to}]
	if !ok {
		return clicks.Carryover{}, fmt.Errorf("%w: %s to %s", embedded_board.ErrNoCarryover, from, to)
	}
	return carry, nil
}

type world struct {
	useCase *switch_board_usecase.UseCase
	tiles   *inmemory_tile_storage.Storage
	eras    *eras.MemoryStore
	ledger  *inmemory_ledger_storage.Storage
	stored  *inmemory_tile_storage.MemoryPersistence
}

func newWorld(t *testing.T) world {
	t.Helper()
	ctx := t.Context()
	grounds := map[clicks.BoardName]*clicks.Ground{
		"earth-1": clicks.NamedGround("earth-1", clicks.BordersOf(3), clicks.NewCapitals(nil)),
		"earth-2": clicks.NamedGround("earth-2", clicks.BordersOf(8), clicks.NewCapitals(nil)),
		"siege-1": clicks.NamedGround("siege-1", clicks.BordersOf(2), clicks.NewCapitals(nil)),
	}
	boards, err := eras.BoardsOf(eras.Shipped, eras.Earth1Carried, eras.Earth2Carried, eras.Siege1Carried)
	require.NoError(t, err)
	carried := eras.NewGrounds(boards, func(name clicks.BoardName) (*clicks.Ground, error) { return grounds[name], nil })
	stored := inmemory_tile_storage.NewMemoryPersistence(map[uint32]string{1: "fr", 2: "de"})
	stored.Shield(1, 5)
	tiles := inmemory_tile_storage.New(carried, clicks.NewPlayed(), clicks.NoCapitals(), inmemory_tile_storage.Config{},
		stored, cpatomic.FakeTransactioner{}, &inmemory_tile_storage.RecordedPlacements{}, cptime.SystemClock{},
		slog.New(slog.DiscardHandler))
	require.NoError(t, tiles.Load(ctx, clicks.World.On(eras.Earth1.ID)))
	carry, err := clicks.NewCarryover("earth-1", "earth-2", 3, parents)
	require.NoError(t, err)
	kept := eras.NewMemoryStore()
	takes := inmemory_ledger_storage.New(inmemory_ledger_storage.Config{}, inmemory_ledger_storage.NewMemoryPersistence(),
		slog.New(slog.DiscardHandler))
	require.NoError(t, takes.Load(ctx))
	return world{
		useCase: switch_board_usecase.New(tiles, carried, carryovers{{"earth-1", "earth-2"}: carry}, kept, takes,
			cptime.NewFixedClock(now)),
		tiles: tiles, eras: kept, ledger: takes, stored: stored,
	}
}

func (w world) events() []ledger.Event {
	var events []ledger.Event
	w.ledger.Replay(func(on clicks.Stage, event ledger.Event) {
		if on != clicks.World.On(eras.Earth2.ID) {
			panic(fmt.Sprintf("an event on %v", on))
		}
		events = append(events, event)
	})
	return events
}

func TestASwitchCarriesTheWorldOntoTheNewBoardStartsItsEraAndRecordsNoTake(t *testing.T) {
	w := newWorld(t)

	out, err := w.useCase.Execute(t.Context(), switch_board_usecase.Command{Map: clicks.World, Board: "earth-2"})

	require.NoError(t, err)
	assert.Equal(t, now, out.SwitchedAt)
	assert.Equal(t, clicks.Carried{
		From: "earth-1", To: "earth-2", Tiles: 8, Carried: 7, Blank: 1, HeldBefore: 2, HeldAfter: 6,
		ShieldsBefore: 5, ShieldsAfter: 5,
	}, out.Carried)
	assert.Equal(t, clicks.World.On(eras.Earth2.ID), w.tiles.Stage())
	era, err := w.eras.Latest(t.Context(), clicks.World)
	require.NoError(t, err)
	assert.Equal(t, eras.Era{Map: clicks.World, Board: eras.Earth2, StartedAt: now}, era)
	assert.Equal(t, []ledger.Event{ledger.Switching{From: "earth-1", To: "earth-2", FromTiles: 3, ToTiles: 8, At: now}},
		w.events(), "one board_switched on the new board's stage")
	assert.Equal(t, map[uint32]string{1: "fr", 2: "fr", 3: "fr", 4: "fr", 5: "de", 6: "de"}, w.stored.Stored())
	assert.Equal(t, map[uint32]int{1: 2, 2: 1, 3: 1, 4: 1}, w.stored.StoredShields())
}

func TestADryRunCountsTheSwitchAndChangesNothing(t *testing.T) {
	w := newWorld(t)

	out, err := w.useCase.Execute(t.Context(), switch_board_usecase.Command{Map: clicks.World, Board: "earth-2", DryRun: true})

	require.NoError(t, err)
	assert.True(t, out.SwitchedAt.IsZero())
	assert.Equal(t, uint32(6), out.Carried.HeldAfter)
	assert.Equal(t, clicks.World.On(eras.Earth1.ID), w.tiles.Stage())
	era, err := w.eras.Latest(t.Context(), clicks.World)
	require.NoError(t, err)
	assert.Equal(t, eras.Earth1, era.Board)
	assert.Empty(t, w.events())
	assert.Equal(t, map[uint32]string{1: "fr", 2: "de"}, w.stored.Stored())
}

func TestASwitchIsRefusedWhereItCannotGo(t *testing.T) {
	for name, test := range map[string]struct {
		command switch_board_usecase.Command
		refused error
	}{
		"another map than the world's": {
			command: switch_board_usecase.Command{Map: "battle", Board: "earth-2"}, refused: switch_board_usecase.ErrNotTheWorld,
		},
		"a board this process does not carry": {
			command: switch_board_usecase.Command{Map: clicks.World, Board: "earth-3"}, refused: eras.ErrNotCarried,
		},
		"the board the world plays": {
			command: switch_board_usecase.Command{Map: clicks.World, Board: "earth-1"}, refused: switch_board_usecase.ErrPlaysAlready,
		},
		"a board with no carry-over to it": {
			command: switch_board_usecase.Command{Map: clicks.World, Board: "siege-1"}, refused: embedded_board.ErrNoCarryover,
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, dry := range []bool{true, false} {
				w := newWorld(t)
				test.command.DryRun = dry

				_, err := w.useCase.Execute(t.Context(), test.command)

				require.ErrorIs(t, err, test.refused)
				assert.Equal(t, clicks.World.On(eras.Earth1.ID), w.tiles.Stage())
				assert.Empty(t, w.events())
			}
		})
	}
}

func TestASwitchIsRefusedWhileAnArenaPlays(t *testing.T) {
	w := newWorld(t)
	_, err := w.tiles.Blank(t.Context(), clicks.MapID("battle").On(eras.Earth1.ID), nil, nil,
		func(context.Context, clicks.Stage) (bool, error) { return true, nil })
	require.NoError(t, err)

	for _, dry := range []bool{true, false} {
		_, err := w.useCase.Execute(t.Context(), switch_board_usecase.Command{Map: clicks.World, Board: "earth-2", DryRun: dry})
		require.ErrorIs(t, err, switch_board_usecase.ErrArenaPlays)
	}
	era, err := w.eras.Latest(t.Context(), clicks.World)
	require.NoError(t, err)
	assert.Equal(t, eras.Earth1, era.Board)
}

func TestASecondSwitchIsRefused(t *testing.T) {
	w := newWorld(t)
	_, err := w.useCase.Execute(t.Context(), switch_board_usecase.Command{Map: clicks.World, Board: "earth-2"})
	require.NoError(t, err)

	_, err = w.useCase.Execute(t.Context(), switch_board_usecase.Command{Map: clicks.World, Board: "earth-2"})

	require.ErrorIs(t, err, switch_board_usecase.ErrPlaysAlready)
	assert.Len(t, w.events(), 1)
}
