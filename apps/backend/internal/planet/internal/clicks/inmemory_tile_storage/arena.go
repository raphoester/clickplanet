package inmemory_tile_storage

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Blank hands keep a copy of the whole map memory holds (the tiles not flushed yet too), then empties the map as Wipe
// does, in keep's transaction, once keep says to.
func (s *Storage) Blank(ctx context.Context, keep func(ctx context.Context, copied clicks.MapCopy) (bool, error)) (bool, error) {
	s.flushing.Lock()
	defer s.flushing.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	copied := s.copyLocked()
	blanked, err := s.persistence.Wipe(ctx, func(ctx context.Context) (bool, error) { return keep(ctx, copied) })
	if err != nil {
		return false, fmt.Errorf("failed to blank the stored map: %w", err)
	}
	if !blanked {
		return false, nil
	}

	s.emptyLocked()
	s.restartLocked()

	return true, nil
}

// PutBack makes the map the copy take answers, in take's transaction, once take says to: stored, then read into
// memory as the boot reads it. The map starts a new epoch and every stream is cut. What memory held before is
// dropped, flushed or not.
func (s *Storage) PutBack(ctx context.Context, take func(ctx context.Context) (clicks.MapCopy, bool, error)) (bool, error) {
	s.flushing.Lock()
	defer s.flushing.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		tiles      []Tile
		landmasses []Landmass
	)
	put, err := s.persistence.Replace(ctx, func(ctx context.Context) ([]Tile, []Landmass, bool, error) {
		taken, ok, err := take(ctx)
		tiles, landmasses = tilesOf(taken), landmassesOf(taken)
		return tiles, landmasses, ok, err
	})
	if err != nil {
		return false, fmt.Errorf("failed to put the stored map back: %w", err)
	}
	if !put {
		return false, nil
	}

	s.emptyLocked()
	defer s.restartLocked()
	if err := s.fillLocked(ctx, visitTiles(tiles), visitLandmasses(landmasses)); err != nil {
		return true, fmt.Errorf("put the map back in postgres, but not in memory: %w", err)
	}

	return true, nil
}

func (s *Storage) copyLocked() clicks.MapCopy {
	var copied clicks.MapCopy
	for tile, state := range s.board.tiles {
		if state.owner == unownedCode {
			continue
		}
		copied.Tiles = append(copied.Tiles, clicks.TileCopy{
			ID: uint32(tile), Owner: s.board.nameOf(state.owner), Shields: state.shields, //nolint:gosec // a tile id.
			Holder: s.board.holders.accountOf(state.holder),
		})
	}
	s.landmasses.each(func(landmass clicks.LandmassID) {
		if flag := s.landmasses.lockedTo(landmass); flag != unownedCode {
			copied.Landmasses = append(copied.Landmasses, clicks.LandmassCopy{
				ID: landmass, Asset: s.landmasses.asset(), FortifiedBy: s.board.nameOf(flag),
			})
		}
	})
	return copied
}

func tilesOf(copied clicks.MapCopy) []Tile {
	tiles := make([]Tile, 0, len(copied.Tiles))
	for _, tile := range copied.Tiles {
		tiles = append(tiles, Tile{ID: tile.ID, Owner: tile.Owner, Shields: int(tile.Shields), Holder: tile.Holder})
	}
	return tiles
}

func landmassesOf(copied clicks.MapCopy) []Landmass {
	landmasses := make([]Landmass, 0, len(copied.Landmasses))
	for _, landmass := range copied.Landmasses {
		landmasses = append(landmasses, Landmass{ID: landmass.ID, Asset: landmass.Asset, FortifiedBy: landmass.FortifiedBy})
	}
	return landmasses
}

func visitTiles(tiles []Tile) func(context.Context, func(Tile)) error {
	return func(_ context.Context, visit func(Tile)) error {
		for _, tile := range tiles {
			visit(tile)
		}
		return nil
	}
}

func visitLandmasses(landmasses []Landmass) func(context.Context, string, func(clicks.LandmassID, string)) error {
	return func(_ context.Context, asset string, visit func(clicks.LandmassID, string)) error {
		for _, landmass := range landmasses {
			if landmass.Asset == asset {
				visit(landmass.ID, landmass.FortifiedBy)
			}
		}
		return nil
	}
}
