//go:build testing

package inmemory_unlock_store

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/unlocks"
)

type Store struct {
	mu       sync.Mutex
	kept     map[bonuses.Account]unlocks.Unlocks
	failWith error
}

var _ unlocks.Store = (*Store)(nil)

func New() *Store {
	return &Store{kept: map[bonuses.Account]unlocks.Unlocks{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failWith = err
}

func (s *Store) Change(_ context.Context, account bonuses.Account, change func(unlocks.Unlocks) unlocks.Unlocks) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failWith != nil {
		return s.failWith
	}
	s.kept[account] = change(s.kept[account])
	return nil
}

func (s *Store) Unlocked(_ context.Context, account bonuses.Account) (bonuses.Unlocked, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failWith != nil {
		return nil, s.failWith
	}
	return s.kept[account].TakeSnapshot().Kinds, nil
}

func (s *Store) Forget(_ context.Context, account bonuses.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failWith != nil {
		return s.failWith
	}
	delete(s.kept, account)
	return nil
}
