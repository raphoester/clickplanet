package inmemory_tile_storage

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// Persistence keeps each map's tiles and fortify locks: the world's, and each arena's.
type Persistence interface {
	Load(ctx context.Context, played clicks.MapID, visit func(tile Tile)) error
	LoadLandmasses(ctx context.Context, played clicks.MapID, asset string,
		visit func(landmass clicks.LandmassID, fortifiedBy string)) error
	Save(ctx context.Context, played clicks.MapID, tiles []Tile, landmasses []Landmass) error
	// Wipe deletes every tile and every landmass's lock of a map, in one transaction with keep, once keep says to.
	Wipe(ctx context.Context, played clicks.MapID, keep func(ctx context.Context) (bool, error)) (bool, error)
	// ForgetHolder takes an account off every tile of every map.
	ForgetHolder(ctx context.Context, account cpsession.AccountID) error
}

type Tile struct {
	ID      uint32
	Owner   string
	Shields int
	// NoAccount when no player's take holds it: an operator's write, or a take older than the holders.
	Holder cpsession.AccountID
}

// Asset is the borders blob the id indexes: a new map numbers its landmasses again.
type Landmass struct {
	ID          clicks.LandmassID
	Asset       string
	FortifiedBy string
}

const flushTimeout = 10 * time.Second

// Load reads the map that plays: the world's, or the open arena's.
func (s *Storage) Load(ctx context.Context, playing clicks.MapID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.playing = playing
	return s.fillLocked(ctx)
}

// fillLocked puts the tiles and the locks the map in play keeps on the board, as on a blank one, and settles every
// landmass.
func (s *Storage) fillLocked(ctx context.Context) error {
	var (
		owned, outside int
		internErr      error
	)
	err := s.persistence.Load(ctx, s.playing, func(tile Tile) {
		if tile.ID > s.board.last() {
			outside++
			return
		}
		id, err := s.board.intern(tile.Owner)
		if err != nil {
			internErr = err
			return
		}
		s.board.restore(tile.ID, tileState{
			owner:   id,
			shields: uint8(min(max(tile.Shields, 0), math.MaxUint8)), //nolint:gosec // clamped to a byte.
			holder:  s.board.holders.intern(tile.Holder),
		})
		s.landmasses.moved(tile.ID, unownedCode, id)
		owned++
	})
	if err != nil {
		return fmt.Errorf("failed to read stored tiles: %w", err)
	}
	if internErr != nil {
		return fmt.Errorf("failed to intern a stored country: %w", internErr)
	}

	fortified, strays := 0, 0
	err = s.persistence.LoadLandmasses(ctx, s.playing, s.landmasses.asset(), func(landmass clicks.LandmassID, fortifiedBy string) {
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

	s.logger.Info("loaded the tile map", slog.String("map", string(s.playing)), slog.Int("ownedTiles", owned), slog.Int("countryCodes", s.board.countries()),
		slog.Int("fortifiedLandmasses", fortified))

	return nil
}

func (s *Storage) Name() string { return "tiles-storage" }

func (s *Storage) Run(ctx context.Context) {
	var ticking sync.WaitGroup
	ticking.Go(func() { s.publishTallyEvery(ctx, tallyInterval) })
	defer ticking.Wait()

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
	s.flushing.Lock()
	defer s.flushing.Unlock()

	s.mu.Lock()
	playing, tiles, landmasses := s.playing, s.board.drainDirty(), s.landmasses.drainDirty(s.board.nameOf)
	s.mu.Unlock()

	if err := s.save(ctx, playing, tiles, landmasses); err != nil {
		s.mu.Lock()
		s.markDirtyLocked(tiles, landmasses)
		s.mu.Unlock()
		return err
	}
	return nil
}

// flushLocked saves what changed while the caller holds the flush lock and the tiles lock.
func (s *Storage) flushLocked(ctx context.Context) error {
	tiles, landmasses := s.board.drainDirty(), s.landmasses.drainDirty(s.board.nameOf)
	if err := s.save(ctx, s.playing, tiles, landmasses); err != nil {
		s.markDirtyLocked(tiles, landmasses)
		return err
	}
	return nil
}

func (s *Storage) save(ctx context.Context, playing clicks.MapID, tiles []Tile, landmasses []Landmass) error {
	if len(tiles) == 0 && len(landmasses) == 0 {
		return nil
	}
	if err := s.persistence.Save(ctx, playing, tiles, landmasses); err != nil {
		return fmt.Errorf("failed to save %d tiles and %d landmasses: %w", len(tiles), len(landmasses), err)
	}
	return nil
}

func (s *Storage) markDirtyLocked(tiles []Tile, landmasses []Landmass) {
	for _, tile := range tiles {
		s.board.markDirty(tile.ID)
	}
	for _, landmass := range landmasses {
		s.landmasses.markDirty(landmass.ID)
	}
}
