package inmemory_tile_storage

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

const maxCodes = math.MaxUint16 + 1

const unownedCode = uint16(0)

func New(
	maxIndex uint32,
	config Config,
	persistence Persistence,
	logger *slog.Logger,
) *Storage {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	config = config.withDefaults()

	s := &Storage{
		config:      config,
		logger:      logger,
		persistence: persistence,
		maxIndex:    maxIndex,
		tiles:       make([]tileState, int(maxIndex)+1),
		dirty:       make([]uint64, (int(maxIndex)+64)/64),
		counts:      []uint32{0},
		codes:       []string{""},
		codeIDs:     map[string]uint16{"": unownedCode},
		subscribers: cpcolls.NewSet[*subscriber](),
	}

	return s
}

type Storage struct {
	config      Config
	logger      *slog.Logger
	persistence Persistence
	maxIndex    uint32

	tilesMu sync.RWMutex
	tiles   []tileState
	counts  []uint32
	codes   []string
	codeIDs map[string]uint16
	dirty   []uint64

	subscribersMu sync.Mutex
	subscribers   *cpcolls.Set[*subscriber]
}

type subscriber struct {
	ch      chan clicks.Change
	dropped atomic.Uint64
}

func (s *Storage) Set(_ context.Context, tile uint32, value string) error {
	return s.put(clicks.TileUpdate{Tile: tile, Value: value})
}

func (s *Storage) Click(_ context.Context, tile uint32, value string) error {
	return s.put(clicks.TileUpdate{Tile: tile, Value: value, Clicked: true})
}

func (s *Storage) put(update clicks.TileUpdate) error {
	if update.Tile > s.maxIndex {
		return fmt.Errorf("tile %d out of range (max %d)", update.Tile, s.maxIndex)
	}

	previous, changed, err := s.set(update.Tile, update.Value)
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	update.Previous = previous
	s.publish(clicks.Change{Update: &update})

	return nil
}

func (s *Storage) Clear(_ context.Context, blast clicks.Blast) (clicks.Blast, error) {
	cleared := make([]uint32, 0, len(blast.Cleared))
	var struck []uint32

	s.tilesMu.Lock()
	for _, tile := range blast.Cleared {
		if tile > s.maxIndex {
			s.tilesMu.Unlock()
			return clicks.Blast{}, fmt.Errorf("tile %d out of range (max %d)", tile, s.maxIndex)
		}
		if s.tiles[tile].shields > 0 {
			s.tiles[tile].shields--
			s.markDirtyLocked(tile)
			struck = append(struck, tile)
			continue
		}
		if s.tiles[tile].owner != unownedCode {
			s.counts[s.tiles[tile].owner]--
			s.tiles[tile] = ownedBy(unownedCode)
			s.markDirtyLocked(tile)
			cleared = append(cleared, tile)
		}
	}
	s.tilesMu.Unlock()

	blast.Cleared = cleared
	blast.Struck = struck
	s.publish(clicks.Change{Blast: &blast})

	return blast, nil
}

func (s *Storage) Share(country string) float64 {
	s.tilesMu.RLock()
	defer s.tilesMu.RUnlock()

	id, ok := s.codeIDs[country]
	if !ok || id == unownedCode {
		return 0
	}

	return float64(s.counts[id]) / float64(s.maxIndex)
}

func (s *Storage) Owner(tile uint32) (string, bool) {
	if tile > s.maxIndex {
		return "", false
	}

	s.tilesMu.RLock()
	defer s.tilesMu.RUnlock()

	return s.codes[s.tiles[tile].owner], true
}

func (s *Storage) set(tile uint32, value string) (previous string, changed bool, err error) {
	s.tilesMu.Lock()
	defer s.tilesMu.Unlock()

	previous = s.codes[s.tiles[tile].owner]
	if previous == value {
		return previous, false, nil
	}

	id, err := s.internLocked(value)
	if err != nil {
		return "", false, err
	}

	if s.tiles[tile].owner != unownedCode {
		s.counts[s.tiles[tile].owner]--
	}
	if id != unownedCode {
		s.counts[id]++
	}
	s.tiles[tile] = ownedBy(id)
	s.markDirtyLocked(tile)

	return previous, true, nil
}

func (s *Storage) internLocked(value string) (uint16, error) {
	if id, ok := s.codeIDs[value]; ok {
		return id, nil
	}

	if len(s.codes) >= maxCodes {
		return 0, fmt.Errorf("country code table is full (%d entries)", maxCodes)
	}

	id := uint16(len(s.codes))
	s.codes = append(s.codes, value)
	s.counts = append(s.counts, 0)
	s.codeIDs[value] = id

	return id, nil
}

func (s *Storage) Subscribe(ctx context.Context) (<-chan clicks.Change, error) {
	sub := &subscriber{ch: make(chan clicks.Change, s.config.SubscriberBuffer)}

	s.subscribersMu.Lock()
	s.subscribers.Add(sub)
	s.subscribersMu.Unlock()

	go func() {
		<-ctx.Done()

		s.subscribersMu.Lock()
		defer s.subscribersMu.Unlock()

		s.subscribers.Delete(sub)
		close(sub.ch)
	}()

	return sub.ch, nil
}

const dropLogInterval = 1000

func (s *Storage) publish(change clicks.Change) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	s.subscribers.ForEach(func(sub *subscriber) {
		select {
		case sub.ch <- change:
		default:
			dropped := sub.dropped.Add(1)
			if dropped == 1 || dropped%dropLogInterval == 0 {
				s.logger.Warn("dropped map change for a slow subscriber",
					slog.Uint64("tile", uint64(tileOf(change))),
					slog.Uint64("droppedTotal", dropped),
				)
			}
		}
	})
}

func tileOf(change clicks.Change) uint32 {
	if change.Blast != nil {
		return change.Blast.Tile
	}

	return change.Update.Tile
}

func (s *Storage) DroppedUpdates() uint64 {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	var total uint64
	s.subscribers.ForEach(func(sub *subscriber) {
		total += sub.dropped.Load()
	})

	return total
}

var _ clicks.TileStorage = (*Storage)(nil)

func (s *Storage) Shields(tile uint32) int {
	if tile > s.maxIndex {
		return 0
	}

	s.tilesMu.RLock()
	defer s.tilesMu.RUnlock()

	return int(s.tiles[tile].shields)
}

func (s *Storage) Strike(_ context.Context, tile uint32, owner string) bool {
	if tile > s.maxIndex {
		return false
	}

	s.tilesMu.Lock()
	if s.codes[s.tiles[tile].owner] != owner || s.tiles[tile].shields == 0 {
		s.tilesMu.Unlock()
		return false
	}
	s.tiles[tile].shields--
	s.markDirtyLocked(tile)
	left := int(s.tiles[tile].shields)
	s.tilesMu.Unlock()

	s.publish(clicks.Change{Update: &clicks.TileUpdate{Tile: tile, Value: owner, Previous: owner, Shields: left}})

	return true
}

func (s *Storage) Shield(_ context.Context, tile uint32, country string, most int) error {
	if tile > s.maxIndex {
		return fmt.Errorf("%w: %d", clicks.ErrTileOutOfRange, tile)
	}

	s.tilesMu.Lock()
	owner := s.codes[s.tiles[tile].owner]
	if err := clicks.ShieldError(owner, country, int(s.tiles[tile].shields), min(most, math.MaxUint8)); err != nil {
		s.tilesMu.Unlock()
		return fmt.Errorf("tile %d: %w", tile, err)
	}
	s.tiles[tile].shields++
	s.markDirtyLocked(tile)
	now := int(s.tiles[tile].shields)
	s.tilesMu.Unlock()

	s.publish(clicks.Change{Update: &clicks.TileUpdate{Tile: tile, Value: country, Previous: country, Shields: now}})

	return nil
}
