// Package inmemory_ledger_storage keeps the ledger in memory only: a restart forgets it, the way it forgets the throttle.
package inmemory_ledger_storage

import (
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

func New() *Storage {
	return &Storage{tiles: make(map[uint32]ledger.Taking)}
}

type Storage struct {
	mu    sync.Mutex
	tiles map[uint32]ledger.Taking
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
}

func (s *Storage) PaintedWith(country string) []ledger.Taking {
	return s.collect(func(taking ledger.Taking) bool { return taking.Country == country })
}

func (s *Storage) TakenBy(scope string) []ledger.Taking {
	return s.collect(func(taking ledger.Taking) bool { return taking.Scope == scope })
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
		}
	}
}

func (s *Storage) ForgetBefore(cutoff time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for tile, taking := range s.tiles {
		if taking.At.Before(cutoff) {
			delete(s.tiles, tile)
		}
	}
}
