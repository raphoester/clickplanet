package memory_tile_storage

import (
	"context"
	"fmt"
	"log/slog"
	"math/bits"
	"time"
)

// Persistence is where the map is kept between boots. It is never read after Load.
type Persistence interface {
	Load(ctx context.Context, visit func(tile uint32, owner string)) error
	Save(ctx context.Context, tiles []uint32, owners []string) error
}

const flushTimeout = 10 * time.Second

// Load refuses rather than start empty: an empty map that then flushes would be every player's territory gone.
func (s *Storage) Load(ctx context.Context) error {
	s.tilesMu.Lock()
	defer s.tilesMu.Unlock()

	var (
		owned, outside int
		internErr      error
	)
	err := s.persistence.Load(ctx, func(tile uint32, owner string) {
		if tile > s.maxIndex {
			outside++
			return
		}
		id, err := s.internLocked(owner)
		if err != nil {
			internErr = err
			return
		}
		s.tiles[tile] = id
		s.counts[id]++
		owned++
	})
	if err != nil {
		return fmt.Errorf("failed to read stored tiles: %w", err)
	}
	if internErr != nil {
		return fmt.Errorf("failed to intern a stored country: %w", internErr)
	}

	if outside > 0 {
		s.logger.Warn("stored tiles past the end of the map were ignored", slog.Int("tiles", outside))
	}

	if owned == 0 {
		return s.importLegacySnapshotLocked()
	}

	if s.config.LegacySnapshotPath != "" && fileExists(s.config.LegacySnapshotPath) {
		s.logger.Warn("a legacy tile snapshot is still on disk but postgres already holds the map, ignoring it",
			slog.String("path", s.config.LegacySnapshotPath))
	}

	s.logger.Info("loaded the tile map", slog.Int("ownedTiles", owned), slog.Int("countryCodes", len(s.codes)-1))

	return nil
}

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

// Flush writes every tile changed since the last flush, as it is now. A failed write keeps them marked.
func (s *Storage) Flush(ctx context.Context) error {
	tiles, owners := s.takeDirty()
	if len(tiles) == 0 {
		return nil
	}

	if err := s.persistence.Save(ctx, tiles, owners); err != nil {
		s.tilesMu.Lock()
		for _, tile := range tiles {
			s.markDirtyLocked(tile)
		}
		s.tilesMu.Unlock()

		return fmt.Errorf("failed to save %d tiles: %w", len(tiles), err)
	}

	s.retireLegacySnapshot()

	return nil
}

func (s *Storage) takeDirty() ([]uint32, []string) {
	s.tilesMu.Lock()
	defer s.tilesMu.Unlock()

	var (
		tiles  []uint32
		owners []string
	)
	for w, word := range s.dirty {
		if word == 0 {
			continue
		}
		s.dirty[w] = 0
		for word != 0 {
			tile := uint32(w*64 + bits.TrailingZeros64(word)) //nolint:gosec // tile <= maxIndex, which is a uint32.
			word &= word - 1
			tiles = append(tiles, tile)
			owners = append(owners, s.codes[s.tiles[tile]])
		}
	}

	return tiles, owners
}

func (s *Storage) markDirtyLocked(tile uint32) {
	s.dirty[tile/64] |= 1 << (tile % 64)
}
