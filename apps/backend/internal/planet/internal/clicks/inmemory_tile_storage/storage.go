package inmemory_tile_storage

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

const maxCodes = math.MaxUint16 + 1

const unownedCode = uint16(0)

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
	maxIndex := borders.Tiles()

	s := &Storage{
		config:          config,
		logger:          logger,
		persistence:     persistence,
		maxIndex:        maxIndex,
		borders:         borders,
		tiles:           make([]tileState, int(maxIndex)+1),
		dirty:           make([]uint64, (int(maxIndex)+64)/64),
		counts:          []uint32{0},
		codes:           []string{""},
		codeIDs:         map[string]uint16{"": unownedCode},
		landmassHeld:    make([][]uint32, borders.Landmasses()),
		fortifiedBy:     make([]uint16, borders.Landmasses()),
		dirtyLandmasses: make([]uint64, (borders.Landmasses()+63)/64),
		subscribers:     cpcolls.NewSet[chan clicks.Change](),
	}

	return s
}

type Storage struct {
	config      Config
	logger      *slog.Logger
	persistence Persistence
	maxIndex    uint32
	borders     *clicks.Borders

	tilesMu sync.RWMutex
	tiles   []tileState
	counts  []uint32
	codes   []string
	codeIDs map[string]uint16
	dirty   []uint64

	landmassHeld    [][]uint32
	fortifiedBy     []uint16
	dirtyLandmasses []uint64

	subscribersMu sync.Mutex
	subscribers   *cpcolls.Set[chan clicks.Change]
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
	owners := make([]string, 0, len(blast.Cleared))
	var struck []uint32
	var left []int

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
			left = append(left, int(s.tiles[tile].shields))
			continue
		}
		if s.tiles[tile].owner != unownedCode {
			owners = append(owners, s.codes[s.tiles[tile].owner])
			s.moveLocked(tile, unownedCode)
			cleared = append(cleared, tile)
		}
	}
	s.tilesMu.Unlock()

	blast.Cleared = cleared
	blast.Owners = owners
	blast.Struck = struck
	blast.Left = left
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

func (s *Storage) Territories() map[string]uint32 {
	s.tilesMu.RLock()
	defer s.tilesMu.RUnlock()

	held := map[string]uint32{}
	for id, count := range s.counts {
		if id != int(unownedCode) && count > 0 {
			held[s.codes[id]] = count
		}
	}
	return held
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

	s.moveLocked(tile, id)

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
	changes := make(chan clicks.Change, s.config.SubscriberBuffer)

	s.subscribersMu.Lock()
	s.subscribers.Add(changes)
	s.subscribersMu.Unlock()

	go func() {
		<-ctx.Done()

		s.subscribersMu.Lock()
		defer s.subscribersMu.Unlock()

		s.unsubscribe(changes)
	}()

	return changes, nil
}

func (s *Storage) publish(change clicks.Change) {
	s.subscribersMu.Lock()
	defer s.subscribersMu.Unlock()

	var behind []chan clicks.Change
	s.subscribers.ForEach(func(changes chan clicks.Change) {
		select {
		case changes <- change:
		default:
			behind = append(behind, changes)
		}
	})
	for _, changes := range behind {
		s.logger.Warn("cut off a map subscriber that fell behind", slog.Int("buffer", s.config.SubscriberBuffer))
		s.unsubscribe(changes)
	}
}

func (s *Storage) unsubscribe(changes chan clicks.Change) {
	if !s.subscribers.Contains(changes) {
		return
	}

	s.subscribers.Delete(changes)
	close(changes)
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
