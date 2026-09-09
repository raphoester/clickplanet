package memory_tile_storage

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clicks/domain"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/logging/lf"
	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/xtime"
)

// maxCodes is the number of distinct country codes that fit in the interning
// table. Codes are stored as a uint16 id per tile, so the whole map costs
// 2 bytes per tile whatever the codes look like.
const maxCodes = math.MaxUint16 + 1

// unownedCode is the interned id of the empty country code. It is always 0, so
// a freshly allocated tiles slice reads as "nobody owns anything".
const unownedCode = uint16(0)

// New builds an in-memory tile storage holding maxIndex+1 tiles. When the
// config points at a snapshot file, the state is restored from it — a missing
// or unreadable snapshot only logs, it never prevents the storage from
// starting.
func New(
	maxIndex uint32,
	config Config,
	timeProvider xtime.Provider,
	logger logging.Logger,
) *Storage {
	if logger == nil {
		logger = logging.NewNopLogger()
	}
	if timeProvider == nil {
		timeProvider = xtime.ActualProvider{}
	}

	config = config.withDefaults()

	s := &Storage{
		config:       config,
		logger:       logger,
		timeProvider: timeProvider,
		maxIndex:     maxIndex,
		tiles:        make([]uint16, int(maxIndex)+1),
		codes:        []string{""},
		codeIDs:      map[string]uint16{"": unownedCode},
		updates:      newRing(config.PastUpdatesBuffer),
		subscribers:  make(map[*subscriber]struct{}),
	}

	s.restore()

	return s
}

type Storage struct {
	config       Config
	logger       logging.Logger
	timeProvider xtime.Provider
	maxIndex     uint32

	// tilesMu guards the tiles slice and its interning table.
	tilesMu sync.RWMutex
	tiles   []uint16 // indexed by tile id, holds an interned country code id
	codes   []string // interned country codes, codes[unownedCode] is ""
	codeIDs map[string]uint16
	dirty   bool

	updatesMu sync.Mutex
	updates   *ring

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

	// Setting a tile to the value it already holds writes nothing and
	// publishes nothing, so no redundant fan-out reaches the clients.
	if !changed {
		return nil
	}

	update := domain.TileUpdate{Tile: tile, Value: value, Previous: previous}
	s.recordUpdate(update)
	s.publish(update)

	return nil
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

// internLocked returns the id of a country code, adding it to the table if it
// is new. Callers must hold tilesMu for writing.
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

// Subscribe returns a channel fed with every tile update, for as long as ctx
// lives. The channel is closed once ctx is cancelled. Several subscribers can
// coexist: each gets its own buffered channel, and a subscriber that cannot
// keep up has updates dropped rather than stalling every writer.
func (s *Storage) Subscribe(ctx context.Context) (<-chan domain.TileUpdate, error) {
	sub := &subscriber{ch: make(chan domain.TileUpdate, s.config.SubscriberBuffer)}

	s.subscribersMu.Lock()
	s.subscribers[sub] = struct{}{}
	s.subscribersMu.Unlock()

	go func() {
		<-ctx.Done()

		// Removing and closing under the same lock publish() takes means no
		// send can race with the close.
		s.subscribersMu.Lock()
		defer s.subscribersMu.Unlock()

		delete(s.subscribers, sub)
		close(sub.ch)
	}()

	return sub.ch, nil
}

// dropLogInterval keeps a permanently saturated subscriber from flooding the
// logs: the first drop is logged, then one in every dropLogInterval.
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
					lf.Any("tile", update.Tile),
					lf.Any("droppedTotal", dropped),
				)
			}
		}
	}
}

func (s *Storage) recordUpdate(update domain.TileUpdate) {
	now := s.timeProvider.Now()

	s.updatesMu.Lock()
	defer s.updatesMu.Unlock()

	s.updates.evictBefore(now.Add(-s.config.PastUpdatesRetention))
	s.updates.push(timedUpdate{at: now, update: update})
}

// PastUpdates returns the updates recorded in [now-duration, now], oldest
// first. It only sees what still fits in the in-memory ring buffer.
func (s *Storage) PastUpdates(
	_ context.Context,
	duration time.Duration,
	now time.Time,
) ([]domain.TileUpdate, error) {
	start := now.Add(-duration)

	s.updatesMu.Lock()
	defer s.updatesMu.Unlock()

	return s.updates.since(start), nil
}

// DroppedUpdates reports how many updates were dropped across all current
// subscribers, for tests and diagnostics.
func (s *Storage) DroppedUpdates() uint64 {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	var total uint64
	for sub := range s.subscribers {
		total += sub.dropped.Load()
	}

	return total
}
