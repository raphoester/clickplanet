// Package inmemory_allegiance_storage keeps the flag each account and each scope clicks for most. Memory only:
// a restart forgets it, and the next clicks start it again.
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
	return &Storage{
		clock:    clock,
		accounts: make(map[string]clicks.Allegiance),
		scopes:   make(map[string]clicks.Allegiance),
	}
}

type Storage struct {
	clock cptime.Clock

	mu       sync.RWMutex
	accounts map[string]clicks.Allegiance
	scopes   map[string]clicks.Allegiance
}

// Record counts one tile taken for country at at, for the account and for the scope it came from.
func (s *Storage) Record(account string, scope string, country string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if scope != "" {
		s.scopes[scope] = s.scopes[scope].With(country, at)
	}
	if account != "" {
		s.accounts[account] = s.accounts[account].With(country, at)
	}
}

func (s *Storage) OfAccount(account string) clicks.Allegiance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.accounts[account]
}

func (s *Storage) OfScope(scope string) clicks.Allegiance {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.scopes[scope]
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
	faded := func(_ string, allegiance clicks.Allegiance) bool { return allegiance.Faded(now) }

	s.mu.Lock()
	defer s.mu.Unlock()

	maps.DeleteFunc(s.accounts, faded)
	maps.DeleteFunc(s.scopes, faded)
}
