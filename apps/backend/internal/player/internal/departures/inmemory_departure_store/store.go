//go:build testing

package inmemory_departure_store

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/departures"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Store struct {
	mu       sync.Mutex
	kept     map[players.AccountID]departures.Departure
	failWith error
}

var _ departures.Store = (*Store)(nil)

func New() *Store {
	return &Store{kept: map[players.AccountID]departures.Departure{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failWith = err
}

func (s *Store) Departure(_ context.Context, account players.AccountID) (departures.Departure, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failWith != nil {
		return departures.Departure{}, s.failWith
	}
	return s.kept[account], nil
}

func (s *Store) Change(_ context.Context, account players.AccountID, change func(departures.Departure) departures.Departure) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failWith != nil {
		return s.failWith
	}
	s.kept[account] = change(s.kept[account])
	return nil
}

func (s *Store) Forget(_ context.Context, account players.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failWith != nil {
		return s.failWith
	}
	delete(s.kept, account)
	return nil
}
