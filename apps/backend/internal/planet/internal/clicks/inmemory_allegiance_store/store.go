//go:build testing

// Package inmemory_allegiance_store keeps the tallies in a map, for tests that need the store and not postgres.
// AllegianceStorageContractSuite holds it to what postgres_allegiance_store does.
package inmemory_allegiance_store

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func New() *Store {
	return &Store{tallies: map[clicks.AllegianceKey]clicks.Allegiance{}}
}

type Store struct {
	mu      sync.Mutex
	tallies map[clicks.AllegianceKey]clicks.Allegiance
	failing error
}

var _ clicks.AllegianceStorage = (*Store)(nil)

func (s *Store) Allegiances(_ context.Context, keys ...clicks.AllegianceKey) (map[clicks.AllegianceKey]clicks.Allegiance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failing != nil {
		return nil, s.failing
	}

	found := make(map[clicks.AllegianceKey]clicks.Allegiance, len(keys))
	for _, key := range keys {
		if tally, ok := s.tallies[key]; ok {
			found[key] = tally
		}
	}

	return found, nil
}

func (s *Store) SaveAllegiances(_ context.Context, tallies map[clicks.AllegianceKey]clicks.Allegiance) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failing != nil {
		return s.failing
	}

	for key, tally := range tallies {
		s.tallies[key] = tally
	}

	return nil
}

func (s *Store) DeleteAllegiancesBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failing != nil {
		return 0, s.failing
	}

	var deleted int64
	for key, tally := range s.tallies {
		if tally.At().Before(cutoff) {
			delete(s.tallies, key)
			deleted++
		}
	}

	return deleted, nil
}

// FailWith makes every call return err until Heal.
func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failing = err
}

// Heal makes the calls work again after FailWith.
func (s *Store) Heal() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failing = nil
}
