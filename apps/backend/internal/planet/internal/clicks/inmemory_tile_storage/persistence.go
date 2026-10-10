package inmemory_tile_storage

import (
	"context"
	"fmt"
	"log/slog"
	"math"
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
	s.mu.Lock()
	defer s.mu.Unlock()

	var (
		owned, outside int
		internErr      error
	)
	err := s.persistence.Load(ctx, func(tile uint32, owner string, shields int) {
		if tile > s.board.last() {
			outside++
			return
		}
		id, err := s.board.intern(owner)
		if err != nil {
			internErr = err
			return
		}
		s.board.restore(tile, tileState{owner: id, shields: uint8(min(max(shields, 0), math.MaxUint8))}) //nolint:gosec // clamped to a byte.
		s.landmasses.moved(tile, unownedCode, id)
		owned++
	})
	if err != nil {
		return fmt.Errorf("failed to read stored tiles: %w", err)
	}
	if internErr != nil {
		return fmt.Errorf("failed to intern a stored country: %w", internErr)
	}

	fortified, strays := 0, 0
	err = s.persistence.LoadLandmasses(ctx, s.landmasses.asset(), func(landmass clicks.LandmassID, fortifiedBy string) {
		if !s.landmasses.known(landmass) {
			strays++
			return
		}
		id, err := s.board.intern(fortifiedBy)
		if err != nil {
			internErr = err
			return
		}
		s.landmasses.restore(landmass, id)
		fortified++
	})
	if err != nil {
		return fmt.Errorf("failed to read stored landmasses: %w", err)
	}
	if internErr != nil {
		return fmt.Errorf("failed to intern a stored country: %w", internErr)
	}

	s.landmasses.each(s.settleLandmassLocked)

	if outside > 0 {
		s.logger.Warn("stored tiles past the end of the map were ignored", slog.Int("tiles", outside))
	}
	if strays > 0 {
		s.logger.Warn("stored landmasses past the end of the borders were ignored", slog.Int("landmasses", strays))
	}

	s.logger.Info("loaded the tile map", slog.Int("ownedTiles", owned), slog.Int("countryCodes", s.board.countries()),
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
		s.mu.Lock()
		for _, tile := range tiles {
			s.board.markDirty(tile.ID)
		}
		for _, landmass := range landmasses {
			s.landmasses.markDirty(landmass.ID)
		}
		s.mu.Unlock()

		return fmt.Errorf("failed to save %d tiles and %d landmasses: %w", len(tiles), len(landmasses), err)
	}

	return nil
}

func (s *Storage) takeDirty() ([]Tile, []Landmass) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.board.drainDirty(), s.landmasses.drainDirty(s.board.nameOf)
}
