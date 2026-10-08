//go:build testing

package inmemory_front_store

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Store struct {
	mu       sync.Mutex
	tallies  map[players.AccountID]fronts.Tally
	failWith error
}

var _ fronts.Store = (*Store)(nil)

func New() *Store {
	return &Store{tallies: map[players.AccountID]fronts.Tally{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Tally(account players.AccountID) fronts.Tally {
	s.mu.Lock()
	defer s.mu.Unlock()

	return fronts.TallyOf(s.tallies[account].PlaysFor(), s.tallies[account].PlaysAgainst())
}

func (s *Store) RecordTake(_ context.Context, take fronts.Take) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	s.tallies[take.Account()] = s.tallies[take.Account()].WithTake(take)
	return nil
}

func (s *Store) DeleteAccount(_ context.Context, account players.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	delete(s.tallies, account)
	return nil
}
