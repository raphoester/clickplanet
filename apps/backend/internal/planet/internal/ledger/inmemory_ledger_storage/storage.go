package inmemory_ledger_storage

import (
	"log/slog"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type Config struct {
	// Where the ledger is saved. Empty keeps it in memory, where a restart empties it.
	StatePath    string
	SaveInterval time.Duration
}

const defaultSaveInterval = time.Minute

func (c Config) withDefaults() Config {
	if c.SaveInterval <= 0 {
		c.SaveInterval = defaultSaveInterval
	}
	return c
}

func New(config Config, logger *slog.Logger) *Storage {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	return &Storage{
		config: config.withDefaults(),
		logger: logger,
		tiles:  make(map[uint32]ledger.Taking),
	}
}

type Storage struct {
	config Config
	logger *slog.Logger

	mu    sync.Mutex
	tiles map[uint32]ledger.Taking
	dirty bool
}

var _ ledger.Storage = (*Storage)(nil)

func (s *Storage) Last(tile uint32) (ledger.Taking, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	taking, ok := s.tiles[tile]

	return taking, ok
}

func (s *Storage) Put(taking ledger.Taking) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tiles[taking.Tile] = taking
	s.dirty = true
}

func (s *Storage) PaintedWith(country string) []ledger.Taking {
	return s.collect(func(taking ledger.Taking) bool { return taking.Country == country })
}

func (s *Storage) TakenBy(scope string) []ledger.Taking {
	return s.collect(func(taking ledger.Taking) bool { return taking.Scope == scope })
}

func (s *Storage) All() []ledger.Taking {
	return s.collect(func(ledger.Taking) bool { return true })
}

func (s *Storage) collect(keep func(ledger.Taking) bool) []ledger.Taking {
	s.mu.Lock()
	defer s.mu.Unlock()

	var takings []ledger.Taking
	for _, taking := range s.tiles {
		if keep(taking) {
			takings = append(takings, taking)
		}
	}

	return takings
}

func (s *Storage) Forget(takings []ledger.Taking) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, taking := range takings {
		if s.tiles[taking.Tile] == taking {
			delete(s.tiles, taking.Tile)
			s.dirty = true
		}
	}
}

func (s *Storage) ForgetBefore(cutoff time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for tile, taking := range s.tiles {
		if taking.At.Before(cutoff) {
			delete(s.tiles, tile)
			s.dirty = true
		}
	}
}
