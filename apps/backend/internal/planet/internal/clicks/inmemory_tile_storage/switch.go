package inmemory_tile_storage

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Switch plays the map in play on onto's board, every tile carried over from the board it plays (carry), once keep says
// to, in keep's transaction, which is told what is carried:
//   - each new tile takes its parent's flag and holder, and its share of the parent's shields (carry.Share);
//   - each mine moves to its tile's child nearest it (carry.Heir), and one on a tile with no child is lost;
//   - every live virus ends, and every fortify lock is dropped, then each landmass of the new board settled again;
//   - the flags out stay: they name no tile.
//
// What changed is saved first, so every shield placed is confirmed. Once committed, recorded is told what was carried
// while no write runs (a write after it lands on the new board), and memory is filled from the rows carried as the boot
// fills it from postgres, on the new board's ground, which every rule reads from then on. A new epoch, and every stream
// cut, so every page reads the new board.
func (s *Storage) Switch(
	ctx context.Context, onto clicks.Stage, carry clicks.Carryover,
	keep func(ctx context.Context, carried clicks.Carried) (bool, error), recorded func(carried clicks.Carried),
) (clicks.Carried, bool, error) {
	ground, err := s.grounds.GroundOf(onto.Board)
	if err != nil {
		return clicks.Carried{}, false, fmt.Errorf("failed to read the ground of the board to play: %w", err)
	}

	s.staging.Lock()
	defer s.staging.Unlock()

	s.flushing.Lock()
	defer s.flushing.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.playsLocked(onto); err != nil {
		return clicks.Carried{}, false, err
	}
	locked := time.Now()
	if err := s.flushLocked(ctx); err != nil {
		return clicks.Carried{}, false, fmt.Errorf("failed to save the map before carrying it over: %w", err)
	}
	flushed := time.Now()
	carrying, err := s.carryLocked(ground, carry)
	if err != nil {
		return clicks.Carried{}, false, err
	}
	computed := time.Now()
	switched, err := s.persistence.Carry(ctx, onto, carrying.tiles, carrying.mines, func(ctx context.Context) (bool, error) {
		return keep(ctx, carrying.carried)
	})
	if err != nil || !switched {
		return carrying.carried, false, err //nolint:wrapcheck // keep and the store name it.
	}
	committed := time.Now()

	out := s.out.kept()
	s.stage = onto
	s.groundLocked(ground)
	defer s.restartLocked()
	recorded(carrying.carried)
	if err := s.fillFromLocked(ctx, carriedRows{tiles: carrying.tiles, mines: carrying.mines, out: out}); err != nil {
		return carrying.carried, true, fmt.Errorf("switched the map's board, but could not fill it: %w", err)
	}
	s.logger.Info("carried the map onto its new board under the locks", slog.String("board", string(ground.Name())),
		slog.Duration("flush", flushed.Sub(locked)), slog.Duration("carry", computed.Sub(flushed)),
		slog.Duration("transaction", committed.Sub(computed)), slog.Duration("fill", time.Since(committed)),
		slog.Duration("locked", time.Since(locked)), slog.Int("rows", len(carrying.tiles)))

	return carrying.carried, true, nil
}

// Carried is what Switch would carry onto onto's board, and changes nothing.
func (s *Storage) Carried(onto clicks.Stage, carry clicks.Carryover) (clicks.Carried, error) {
	ground, err := s.grounds.GroundOf(onto.Board)
	if err != nil {
		return clicks.Carried{}, fmt.Errorf("failed to read the ground of the board to play: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.playsLocked(onto); err != nil {
		return clicks.Carried{}, err
	}
	carrying, err := s.carryLocked(ground, carry)
	if err != nil {
		return clicks.Carried{}, err
	}
	return carrying.carried, nil
}

// The rows a switch wrote, which memory is filled from as the boot fills it from postgres: no million rows read back.
type carriedRows struct {
	tiles []Tile
	mines []clicks.MineSnapshot
	out   []OutFlag
}

func (c carriedRows) Load(_ context.Context, _ clicks.MapID, visit func(tile Tile)) error {
	for _, tile := range c.tiles {
		visit(tile)
	}
	return nil
}

// A switch keeps no fortify lock: each landmass of the new board is settled again.
func (carriedRows) LoadLandmasses(context.Context, clicks.MapID, string, func(clicks.LandmassID, string)) error {
	return nil
}

func (c carriedRows) LoadOutFlags(_ context.Context, _ clicks.MapID, visit func(out OutFlag)) error {
	for _, out := range c.out {
		visit(out)
	}
	return nil
}

// Every live virus ends.
func (carriedRows) LoadViruses(context.Context, clicks.MapID, func(clicks.VirusSnapshot)) error {
	return nil
}

func (c carriedRows) LoadMines(_ context.Context, _ clicks.MapID, visit func(mine clicks.MineSnapshot)) error {
	for _, mine := range c.mines {
		visit(mine)
	}
	return nil
}

// A switch moves the map that plays, onto another board.
func (s *Storage) playsLocked(onto clicks.Stage) error {
	if onto.Map != s.stage.Map {
		return fmt.Errorf("%w: %s plays, not %s", clicks.ErrNotInPlay, s.stage.Map, onto.Map)
	}
	if onto.Board == s.stage.Board {
		return fmt.Errorf("%w: map %s plays board %d", clicks.ErrPlaysAlready, onto.Map, onto.Board)
	}
	return nil
}

// The rows of the map carried onto a new board, and their counts.
type carrying struct {
	tiles   []Tile
	mines   []clicks.MineSnapshot
	carried clicks.Carried
}

// carryLocked reads the map in play through carry onto ground: carry must lead from the board in play to ground's.
func (s *Storage) carryLocked(ground *clicks.Ground, carry clicks.Carryover) (carrying, error) {
	old := s.played.Ground()
	if old == nil || old.Name() != carry.From() || ground.Name() != carry.To() || old.Tiles() != carry.OldTiles() ||
		ground.Tiles() != carry.Tiles() {
		return carrying{}, fmt.Errorf("%w: it leads from %s to %s, and the map plays %s to go to %s", clicks.ErrCarryover,
			carry.From(), carry.To(), nameOf(old), ground.Name())
	}

	carried := clicks.Carried{From: carry.From(), To: carry.To(), Tiles: carry.Tiles()}
	for tile := uint32(1); tile <= s.board.last(); tile++ {
		state := s.board.tiles[tile]
		held := state.owner != unownedCode
		if held {
			carried.HeldBefore++
		}
		carried.ShieldsBefore += uint32(state.shields)
		if len(carry.Children(tile)) == 0 {
			carried.Dropped++
			if held {
				carried.DroppedHeld++
			}
		}
	}

	tiles := make([]Tile, 0, carried.HeldBefore*carry.Tiles()/max(carry.OldTiles(), 1)+1)
	for child := uint32(1); child <= carry.Tiles(); child++ {
		parent := carry.Parent(child)
		if parent == 0 {
			carried.Blank++
			continue
		}
		carried.Carried++
		state := s.board.tiles[parent]
		if state.owner == unownedCode {
			continue
		}
		shields := carry.Share(child, int(state.shields))
		tiles = append(tiles, Tile{
			ID: child, Owner: s.board.nameOf(state.owner), Shields: shields, Holder: s.board.holders.accountOf(state.holder),
		})
		carried.HeldAfter++
		carried.ShieldsAfter += uint32(shields) //nolint:gosec // a share of a byte.
	}

	var mines []clicks.MineSnapshot
	for _, tile := range slices.Sorted(maps.Keys(s.mines.laid)) {
		heir, ok := carry.Heir(tile, old.Geography(), ground.Geography())
		if !ok {
			carried.MinesLost++
			continue
		}
		mine := s.mines.laid[tile].TakeSnapshot()
		mine.Tile = heir
		mines = append(mines, mine)
		carried.MinesMoved++
	}

	carried.VirusesEnded = uint32(len(s.viruses.live)) //nolint:gosec // a virus a player, at most.
	s.landmasses.each(func(landmass clicks.LandmassID) {
		if s.landmasses.lockedTo(landmass) != unownedCode {
			carried.LocksDropped++
		}
	})

	return carrying{tiles: tiles, mines: mines, carried: carried}, nil
}

func nameOf(ground *clicks.Ground) clicks.BoardName {
	if ground == nil {
		return "no board"
	}
	return ground.Name()
}
