// Package inmemory_allegiance_storage keeps the tallies of who takes tiles for which flag, under opaque keys. Memory
// only: a restart forgets them, and the next takes start them again.
package inmemory_allegiance_storage

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// forgetEvery is how often the faded allegiances go: a memory of days needs no finer sweep.
const forgetEvery = time.Minute

func New(clock cptime.Clock) *Storage {
	return &Storage{clock: clock, tallies: make(map[clicks.AllegianceKey]clicks.Allegiance)}
}

type Storage struct {
	clock cptime.Clock

	mu      sync.RWMutex
	tallies map[clicks.AllegianceKey]clicks.Allegiance
}

// Record counts one tile taken for country at at, in every tally keys names.
func (s *Storage) Record(country string, at time.Time, keys ...clicks.AllegianceKey) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, key := range keys {
		s.tallies[key] = s.tallies[key].With(country, at)
	}
}

func (s *Storage) Allegiance(key clicks.AllegianceKey) clicks.Allegiance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.tallies[key]
}

func (s *Storage) Name() string { return "allegiance-storage" }

func (s *Storage) Run(ctx context.Context) {
	ticker := time.NewTicker(forgetEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.forgetFaded()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Storage) forgetFaded() {
	now := s.clock.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	maps.DeleteFunc(s.tallies, func(_ clicks.AllegianceKey, allegiance clicks.Allegiance) bool { return allegiance.Faded(now) })
}
