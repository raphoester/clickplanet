package inmemory_tile_storage

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

func New(
	borders *clicks.Borders,
	config Config,
	persistence Persistence,
	clock cptime.Clock,
	logger *slog.Logger,
) *Storage {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	config = config.withDefaults()

	return &Storage{
		clock:       clock,
		config:      config,
		logger:      logger,
		persistence: persistence,
		playing:     clicks.World,
		board:       newBoard(borders.Tiles()),
		landmasses:  newLandmasses(borders),
		recount:     newRecount(borders.Landmasses()),
		version:     clicks.MapVersion{Epoch: uint32(clock.Now().Unix())}, //nolint:gosec // seconds fit a uint32 until 2106.
		feed:        newFeed(config.SubscriberBuffer, logger),
	}
}

type Storage struct {
	clock       cptime.Clock
	config      Config
	logger      *slog.Logger
	persistence Persistence

	// Held through a flush, from what it takes to what it saves, so a wipe never meets a save of the map before it.
	flushing sync.Mutex

	// Guards the map in play, the board, the landmasses and the version, which change together, and the feed's order.
	mu         sync.RWMutex
	playing    clicks.MapID
	board      *board
	landmasses *landmasses
	recount    recount
	version    clicks.MapVersion

	feed *feed
}

var _ clicks.TileStorage = (*Storage)(nil)

func (s *Storage) Set(ctx context.Context, tile uint32, value string) error {
	return s.put(clicks.TileUpdate{Tile: tile, Value: value}, holderOf(ctx))
}

func (s *Storage) Click(ctx context.Context, tile uint32, value string) error {
	return s.put(clicks.TileUpdate{Tile: tile, Value: value, Clicked: true}, holderOf(ctx))
}

func (s *Storage) put(update clicks.TileUpdate, holder cpsession.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	written, changed, err := s.putLocked(update, holder)
	if changed {
		s.publishLocked(clicks.Change{Update: &written})
	}

	return err
}

func (s *Storage) putLocked(update clicks.TileUpdate, holder cpsession.AccountID) (clicks.TileUpdate, bool, error) {
	if update.Tile > s.board.last() {
		return clicks.TileUpdate{}, false, fmt.Errorf("tile %d out of range (max %d)", update.Tile, s.board.last())
	}

	update.Previous = s.board.ownerOf(update.Tile)
	if update.Previous == update.Value {
		return clicks.TileUpdate{}, false, nil
	}

	id, err := s.board.intern(update.Value)
	if err != nil {
		return clicks.TileUpdate{}, false, err
	}

	held := noHolder
	if id != unownedCode {
		held = s.board.holders.intern(holder)
	}
	s.moveLocked(update.Tile, id, held)

	return update, true, nil
}

func (s *Storage) moveLocked(tile uint32, to uint16, holder uint32) {
	from := s.board.move(tile, to, holder)
	s.landmasses.moved(tile, from, to)
	s.recount.moved(s.landmasses.of(tile), from, to)
}

// Under s.mu: every stream then gets the changes in the order of their versions.
func (s *Storage) publishLocked(change clicks.Change) {
	s.version.Seq++
	change.Version = s.version
	s.feed.publish(change)
}

func (s *Storage) Clear(_ context.Context, blast clicks.Blast) (clicks.Blast, error) {
	for _, tile := range blast.Cleared {
		if tile > s.board.last() {
			return clicks.Blast{}, fmt.Errorf("tile %d out of range (max %d)", tile, s.board.last())
		}
	}

	cleared := make([]uint32, 0, len(blast.Cleared))
	owners := make([]string, 0, len(blast.Cleared))
	var struck []uint32
	var left []int
	var struckOwners []string

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, tile := range blast.Cleared {
		if s.board.shieldsOn(tile) > 0 {
			struck = append(struck, tile)
			struckOwners = append(struckOwners, s.board.ownerOf(tile))
			left = append(left, s.board.strike(tile))
			continue
		}
		if owner := s.board.ownerOf(tile); owner != "" {
			owners = append(owners, owner)
			s.moveLocked(tile, unownedCode, noHolder)
			cleared = append(cleared, tile)
		}
	}

	blast.Cleared = cleared
	blast.Owners = owners
	blast.Struck = struck
	blast.Left = left
	blast.StruckOwners = struckOwners
	s.publishLocked(clicks.Change{Blast: &blast})

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

func (s *Storage) Owner(tile uint32) (string, bool) {
	if tile > s.board.last() {
		return "", false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.board.ownerOf(tile), true
}

func (s *Storage) Subscribe(ctx context.Context) (<-chan clicks.Change, clicks.MapVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.feed.subscribe(ctx), s.version, nil
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
	s.mu.Lock()
	defer s.mu.Unlock()

	left, struck := s.strikeLocked(tile, owner)
	if struck {
		s.publishLocked(clicks.Change{Update: &clicks.TileUpdate{Tile: tile, Value: owner, Previous: owner, Shields: left}})
	}

	return struck
}

func (s *Storage) strikeLocked(tile uint32, owner string) (int, bool) {
	if tile > s.board.last() || s.board.ownerOf(tile) != owner || s.board.shieldsOn(tile) == 0 {
		return 0, false
	}

	return s.board.strike(tile), true
}

func (s *Storage) Shield(_ context.Context, tile uint32, country string, most int) error {
	if tile > s.board.last() {
		return fmt.Errorf("%w: %d", clicks.ErrTileOutOfRange, tile)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := clicks.ShieldError(s.board.ownerOf(tile), country, s.board.shieldsOn(tile), min(most, math.MaxUint8)); err != nil {
		return fmt.Errorf("tile %d: %w", tile, err)
	}
	now := s.board.raise(tile)
	s.publishLocked(clicks.Change{Update: &clicks.TileUpdate{Tile: tile, Value: country, Previous: country, Shields: now}})

	return nil
}

func (s *Storage) StateBatchDense(start uint32, end uint32) (clicks.DenseBatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	end = min(end, s.board.last())
	if start > end {
		return clicks.DenseBatch{}, fmt.Errorf("invalid tile range %d..%d", start, end)
	}

	batch := s.board.batch(start, end)
	batch.Version = s.version

	return batch, nil
}
