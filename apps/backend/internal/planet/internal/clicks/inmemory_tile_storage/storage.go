package inmemory_tile_storage

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func New(
	borders *clicks.Borders,
	config Config,
	persistence Persistence,
	logger *slog.Logger,
) *Storage {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	config = config.withDefaults()

	return &Storage{
		config:      config,
		logger:      logger,
		persistence: persistence,
		board:       newBoard(borders.Tiles()),
		landmasses:  newLandmasses(borders),
		feed:        newFeed(config.SubscriberBuffer, logger),
	}
}

type Storage struct {
	config      Config
	logger      *slog.Logger
	persistence Persistence

	// Guards the board and the landmasses, which change together.
	mu         sync.RWMutex
	board      *board
	landmasses *landmasses

	feed *feed
}

var _ clicks.TileStorage = (*Storage)(nil)

func (s *Storage) Set(_ context.Context, tile uint32, value string) error {
	return s.put(clicks.TileUpdate{Tile: tile, Value: value})
}

func (s *Storage) Click(_ context.Context, tile uint32, value string) error {
	return s.put(clicks.TileUpdate{Tile: tile, Value: value, Clicked: true})
}

func (s *Storage) put(update clicks.TileUpdate) error {
	if update.Tile > s.board.last() {
		return fmt.Errorf("tile %d out of range (max %d)", update.Tile, s.board.last())
	}

	previous, changed, err := s.set(update.Tile, update.Value)
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	update.Previous = previous
	s.feed.publish(clicks.Change{Update: &update})

	return nil
}

func (s *Storage) set(tile uint32, value string) (previous string, changed bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	previous = s.board.ownerOf(tile)
	if previous == value {
		return previous, false, nil
	}

	id, err := s.board.intern(value)
	if err != nil {
		return "", false, err
	}

	s.moveLocked(tile, id)

	return previous, true, nil
}

func (s *Storage) moveLocked(tile uint32, to uint16) {
	s.landmasses.moved(tile, s.board.move(tile, to), to)
}

func (s *Storage) Clear(_ context.Context, blast clicks.Blast) (clicks.Blast, error) {
	cleared := make([]uint32, 0, len(blast.Cleared))
	owners := make([]string, 0, len(blast.Cleared))
	var struck []uint32
	var left []int

	s.mu.Lock()
	for _, tile := range blast.Cleared {
		if tile > s.board.last() {
			s.mu.Unlock()
			return clicks.Blast{}, fmt.Errorf("tile %d out of range (max %d)", tile, s.board.last())
		}
		if s.board.shieldsOn(tile) > 0 {
			struck = append(struck, tile)
			left = append(left, s.board.strike(tile))
			continue
		}
		if owner := s.board.ownerOf(tile); owner != "" {
			owners = append(owners, owner)
			s.moveLocked(tile, unownedCode)
			cleared = append(cleared, tile)
		}
	}
	s.mu.Unlock()

	blast.Cleared = cleared
	blast.Owners = owners
	blast.Struck = struck
	blast.Left = left
	s.feed.publish(clicks.Change{Blast: &blast})

	return blast, nil
}

func (s *Storage) Share(country string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return float64(s.board.heldBy(country)) / float64(s.board.last())
}

func (s *Storage) Held(country string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return int(s.board.heldBy(country))
}

func (s *Storage) Territories() map[string]uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.board.holdings()
}

func (s *Storage) Owner(tile uint32) (string, bool) {
	if tile > s.board.last() {
		return "", false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.board.ownerOf(tile), true
}

func (s *Storage) Subscribe(ctx context.Context) (<-chan clicks.Change, error) {
	return s.feed.subscribe(ctx), nil
}

func (s *Storage) Shields(tile uint32) int {
	if tile > s.board.last() {
		return 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.board.shieldsOn(tile)
}

func (s *Storage) Strike(_ context.Context, tile uint32, owner string) bool {
	if tile > s.board.last() {
		return false
	}

	s.mu.Lock()
	if s.board.ownerOf(tile) != owner || s.board.shieldsOn(tile) == 0 {
		s.mu.Unlock()
		return false
	}
	left := s.board.strike(tile)
	s.mu.Unlock()

	s.feed.publish(clicks.Change{Update: &clicks.TileUpdate{Tile: tile, Value: owner, Previous: owner, Shields: left}})

	return true
}

func (s *Storage) Shield(_ context.Context, tile uint32, country string, most int) error {
	if tile > s.board.last() {
		return fmt.Errorf("%w: %d", clicks.ErrTileOutOfRange, tile)
	}

	s.mu.Lock()
	if err := clicks.ShieldError(s.board.ownerOf(tile), country, s.board.shieldsOn(tile), min(most, math.MaxUint8)); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("tile %d: %w", tile, err)
	}
	now := s.board.raise(tile)
	s.mu.Unlock()

	s.feed.publish(clicks.Change{Update: &clicks.TileUpdate{Tile: tile, Value: country, Previous: country, Shields: now}})

	return nil
}

func (s *Storage) StateBatchDense(start uint32, end uint32) (clicks.DenseBatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	end = min(end, s.board.last())
	if start > end {
		return clicks.DenseBatch{}, fmt.Errorf("invalid tile range %d..%d", start, end)
	}

	return s.board.batch(start, end), nil
}
