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

// Record counts one click for country, for the payer's account when it has one and for its scope.
func (s *Storage) Record(payer clicks.Payer, country string) {
	now := s.clock.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	if payer.Scope != "" {
		s.scopes[payer.Scope] = s.scopes[payer.Scope].With(country, now)
	}
	if payer.Account != "" {
		s.accounts[payer.Account] = s.accounts[payer.Account].With(country, now)
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
