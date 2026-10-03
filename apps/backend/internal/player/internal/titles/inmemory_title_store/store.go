//go:build testing

package inmemory_title_store

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

type Store struct {
	mu       sync.Mutex
	held     map[players.AccountID]titles.IDs
	worn     map[players.AccountID]titles.ID
	failWith error
}

var _ titles.Store = (*Store)(nil)

func New() *Store {
	return &Store{held: map[players.AccountID]titles.IDs{}, worn: map[players.AccountID]titles.ID{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Held(_ context.Context, account players.AccountID) (titles.IDs, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	return slices.Clone(s.held[account]), nil
}

func (s *Store) Holdings(_ context.Context, accounts []players.AccountID) (titles.Holdings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	holdings := titles.Holdings{}
	for _, account := range accounts {
		if held := s.held[account]; len(held) > 0 {
			holdings[account] = slices.Clone(held)
		}
	}
	return holdings, nil
}

func (s *Store) Grant(_ context.Context, grants titles.Holdings, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	for account, granted := range grants {
		s.held[account] = append(s.held[account], granted.Without(s.held[account])...)
	}
	return nil
}

func (s *Store) Revoke(_ context.Context, revocations titles.Holdings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	for account, revoked := range revocations {
		s.held[account] = s.held[account].Without(revoked)
		if len(s.held[account]) == 0 {
			delete(s.held, account)
		}
	}
	return nil
}

func (s *Store) Worn(_ context.Context, account players.AccountID) (titles.ID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return "", s.failWith
	}
	return s.worn[account], nil
}

func (s *Store) Wear(_ context.Context, account players.AccountID, title titles.ID, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	s.worn[account] = title
	return nil
}

func (s *Store) DeleteAccount(_ context.Context, account players.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	delete(s.held, account)
	delete(s.worn, account)
	return nil
}
