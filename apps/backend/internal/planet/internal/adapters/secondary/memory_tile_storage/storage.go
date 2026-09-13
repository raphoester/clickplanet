package memory_tile_storage

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const maxCodes = math.MaxUint16 + 1

const unownedCode = uint16(0)

func New(
	maxIndex uint32,
	config Config,
	logger *slog.Logger,
) *Storage {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	config = config.withDefaults()

	s := &Storage{
		config:      config,
		logger:      logger,
		maxIndex:    maxIndex,
		tiles:       make([]uint16, int(maxIndex)+1),
		counts:      []uint32{0},
		codes:       []string{""},
		codeIDs:     map[string]uint16{"": unownedCode},
		subscribers: make(map[*subscriber]struct{}),
	}

	s.restore()

	return s
}

type Storage struct {
	config   Config
	logger   *slog.Logger
	maxIndex uint32

	tilesMu sync.RWMutex
	tiles   []uint16
	counts  []uint32
	codes   []string
	codeIDs map[string]uint16
	dirty   bool

	subscribersMu sync.Mutex
	subscribers   map[*subscriber]struct{}
}

type subscriber struct {
	ch      chan clicks.Change
	dropped atomic.Uint64
}

func (s *Storage) Set(_ context.Context, tile uint32, value string) error {
	if tile > s.maxIndex {
		return fmt.Errorf("tile %d out of range (max %d)", tile, s.maxIndex)
	}

	previous, changed, err := s.set(tile, value)
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	s.publish(clicks.Change{Update: &clicks.TileUpdate{Tile: tile, Value: value, Previous: previous}})

	return nil
}

// Clear empties blast.Cleared and publishes the blast once, holding only the tiles that were owned.
func (s *Storage) Clear(_ context.Context, blast clicks.Blast) (clicks.Blast, error) {
	cleared := make([]uint32, 0, len(blast.Cleared))

	s.tilesMu.Lock()
	for _, tile := range blast.Cleared {
		if tile > s.maxIndex {
			s.tilesMu.Unlock()
			return clicks.Blast{}, fmt.Errorf("tile %d out of range (max %d)", tile, s.maxIndex)
		}
		if s.tiles[tile] != unownedCode {
			s.counts[s.tiles[tile]]--
			s.tiles[tile] = unownedCode
			cleared = append(cleared, tile)
		}
	}
	if len(cleared) > 0 {
		s.dirty = true
	}
	s.tilesMu.Unlock()

	blast.Cleared = cleared
	s.publish(clicks.Change{Blast: &blast})

	return blast, nil
}

// Share is the fraction of the whole map a country holds, unowned tiles counted in the whole.
func (s *Storage) Share(country string) float64 {
	s.tilesMu.RLock()
	defer s.tilesMu.RUnlock()

	id, ok := s.codeIDs[country]
	if !ok || id == unownedCode {
		return 0
	}

	return float64(s.counts[id]) / float64(s.maxIndex)
}

// Owner reads one tile; false means past the end of the map, and an unowned tile reads as an empty code.
func (s *Storage) Owner(tile uint32) (string, bool) {
	if tile > s.maxIndex {
		return "", false
	}

	s.tilesMu.RLock()
	defer s.tilesMu.RUnlock()

	return s.codes[s.tiles[tile]], true
}

func (s *Storage) set(tile uint32, value string) (previous string, changed bool, err error) {
	s.tilesMu.Lock()
	defer s.tilesMu.Unlock()

	previous = s.codes[s.tiles[tile]]
	if previous == value {
		return previous, false, nil
	}

	id, err := s.internLocked(value)
	if err != nil {
		return "", false, err
	}

	if s.tiles[tile] != unownedCode {
		s.counts[s.tiles[tile]]--
	}
	if id != unownedCode {
		s.counts[id]++
	}
	s.tiles[tile] = id
	s.dirty = true

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
	s.subscribers[sub] = struct{}{}
	s.subscribersMu.Unlock()

	go func() {
		<-ctx.Done()

		s.subscribersMu.Lock()
		defer s.subscribersMu.Unlock()

		delete(s.subscribers, sub)
		close(sub.ch)
	}()

	return sub.ch, nil
}

const dropLogInterval = 1000

func (s *Storage) publish(change clicks.Change) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for sub := range s.subscribers {
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
	}
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
	for sub := range s.subscribers {
		total += sub.dropped.Load()
	}

	return total
}
