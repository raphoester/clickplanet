package inmemory_tile_storage

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/bits"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Persistence interface {
	Load(ctx context.Context, visit func(tile uint32, owner string, shields int)) error
	LoadLandmasses(ctx context.Context, asset string, visit func(landmass clicks.LandmassID, fortifiedBy string)) error
	Save(ctx context.Context, tiles []Tile, landmasses []Landmass) error
}

type Tile struct {
	ID      uint32
	Owner   string
	Shields int
}

// Asset is the borders blob the id indexes: a new map numbers its landmasses again.
type Landmass struct {
	ID          clicks.LandmassID
	Asset       string
	FortifiedBy string
}

const flushTimeout = 10 * time.Second

func (s *Storage) Load(ctx context.Context) error {
	s.tilesMu.Lock()
	defer s.tilesMu.Unlock()

	var (
		owned, outside int
		internErr      error
	)
	err := s.persistence.Load(ctx, func(tile uint32, owner string, shields int) {
		if tile > s.maxIndex {
			outside++
			return
		}
		id, err := s.internLocked(owner)
		if err != nil {
			internErr = err
			return
		}
		s.holdLocked(tile, tileState{owner: id, shields: uint8(min(max(shields, 0), math.MaxUint8))}) //nolint:gosec // clamped to a byte.
		owned++
	})
	if err != nil {
		return fmt.Errorf("failed to read stored tiles: %w", err)
	}
	if internErr != nil {
		return fmt.Errorf("failed to intern a stored country: %w", internErr)
	}

	fortified, strays := 0, 0
	err = s.persistence.LoadLandmasses(ctx, s.borders.Asset(), func(landmass clicks.LandmassID, fortifiedBy string) {
		if int(landmass) >= len(s.fortifiedBy) {
			strays++
			return
		}
		id, err := s.internLocked(fortifiedBy)
		if err != nil {
			internErr = err
			return
		}
		s.fortifiedBy[landmass] = id
		fortified++
	})
	if err != nil {
		return fmt.Errorf("failed to read stored landmasses: %w", err)
	}
	if internErr != nil {
		return fmt.Errorf("failed to intern a stored country: %w", internErr)
	}

	for landmass := range s.borders.Landmasses() {
		s.settleLandmassLocked(clicks.LandmassID(landmass)) //nolint:gosec // landmasses are uint16 in the blob.
	}

	if outside > 0 {
		s.logger.Warn("stored tiles past the end of the map were ignored", slog.Int("tiles", outside))
	}
	if strays > 0 {
		s.logger.Warn("stored landmasses past the end of the borders were ignored", slog.Int("landmasses", strays))
	}

	s.logger.Info("loaded the tile map", slog.Int("ownedTiles", owned), slog.Int("countryCodes", len(s.codes)-1),
		slog.Int("fortifiedLandmasses", fortified))

	return nil
}

func (s *Storage) Name() string { return "tiles-storage" }

func (s *Storage) Run(ctx context.Context) {
	ticker := time.NewTicker(s.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.flushOrLog(ctx)
		case <-ctx.Done():
			s.flushOrLog(context.WithoutCancel(ctx))
			return
		}
	}
}

func (s *Storage) flushOrLog(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, flushTimeout)
	defer cancel()

	if err := s.Flush(ctx); err != nil {
		s.logger.Error("failed to flush the tile map, retrying next tick", slog.Any("error", err))
	}
}

func (s *Storage) Flush(ctx context.Context) error {
	tiles, landmasses := s.takeDirty()
	if len(tiles) == 0 && len(landmasses) == 0 {
		return nil
	}

	if err := s.persistence.Save(ctx, tiles, landmasses); err != nil {
		s.tilesMu.Lock()
		for _, tile := range tiles {
			s.markDirtyLocked(tile.ID)
		}
		for _, landmass := range landmasses {
			s.markLandmassDirtyLocked(landmass.ID)
		}
		s.tilesMu.Unlock()

		return fmt.Errorf("failed to save %d tiles and %d landmasses: %w", len(tiles), len(landmasses), err)
	}

	return nil
}

func (s *Storage) takeDirty() ([]Tile, []Landmass) {
	s.tilesMu.Lock()
	defer s.tilesMu.Unlock()

	var tiles []Tile
	for w, word := range s.dirty {
		if word == 0 {
			continue
		}
		s.dirty[w] = 0
		for word != 0 {
			tile := uint32(w*64 + bits.TrailingZeros64(word)) //nolint:gosec // tile <= maxIndex, which is a uint32.
			word &= word - 1
			state := s.tiles[tile]
			tiles = append(tiles, Tile{ID: tile, Owner: s.codes[state.owner], Shields: int(state.shields)})
		}
	}

	var landmasses []Landmass
	for w, word := range s.dirtyLandmasses {
		if word == 0 {
			continue
		}
		s.dirtyLandmasses[w] = 0
		for word != 0 {
			landmass := clicks.LandmassID(w*64 + bits.TrailingZeros64(word)) //nolint:gosec // landmasses are uint16.
			word &= word - 1
			landmasses = append(landmasses, Landmass{
				ID: landmass, Asset: s.borders.Asset(), FortifiedBy: s.codes[s.fortifiedBy[landmass]],
			})
		}
	}

	return tiles, landmasses
}

func (s *Storage) markDirtyLocked(tile uint32) {
	s.dirty[tile/64] |= 1 << (tile % 64)
}
