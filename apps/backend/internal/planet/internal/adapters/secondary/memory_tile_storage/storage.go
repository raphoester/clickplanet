package memory_tile_storage

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cplogging"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cplogging/cplf"
)

const maxCodes = math.MaxUint16 + 1

const unownedCode = uint16(0)

func New(
	maxIndex uint32,
	config Config,
	logger cplogging.Logger,
) *Storage {
	if logger == nil {
		logger = cplogging.NewNopLogger()
	}

	config = config.withDefaults()

	s := &Storage{
		config:      config,
		logger:      logger,
		maxIndex:    maxIndex,
		tiles:       make([]uint16, int(maxIndex)+1),
		codes:       []string{""},
		codeIDs:     map[string]uint16{"": unownedCode},
		subscribers: make(map[*subscriber]struct{}),
	}

	s.restore()

	return s
}

type Storage struct {
	config   Config
	logger   cplogging.Logger
	maxIndex uint32

	tilesMu sync.RWMutex
	tiles   []uint16
	codes   []string
	codeIDs map[string]uint16
	dirty   bool

	subscribersMu sync.Mutex
	subscribers   map[*subscriber]struct{}
}

type subscriber struct {
	ch      chan domain.TileUpdate
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

	s.publish(domain.TileUpdate{Tile: tile, Value: value, Previous: previous})

	return nil
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
	s.codeIDs[value] = id

	return id, nil
}

func (s *Storage) Subscribe(ctx context.Context) (<-chan domain.TileUpdate, error) {
	sub := &subscriber{ch: make(chan domain.TileUpdate, s.config.SubscriberBuffer)}

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

func (s *Storage) publish(update domain.TileUpdate) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	for sub := range s.subscribers {
		select {
		case sub.ch <- update:
		default:
			dropped := sub.dropped.Add(1)
			if dropped == 1 || dropped%dropLogInterval == 0 {
				s.logger.Warning("dropped tile update for a slow subscriber",
					cplf.Any("tile", update.Tile),
					cplf.Any("droppedTotal", dropped),
				)
			}
		}
	}
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
