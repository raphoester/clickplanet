//go:build testing

package inmemory_worn_title_store

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/wearing"
)

type Store struct {
	mu       sync.Mutex
	chosen   map[players.AccountID]titles.ID
	failWith error
}

var _ wearing.Store = (*Store)(nil)

func New() *Store {
	return &Store{chosen: map[players.AccountID]titles.ID{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Choice(_ context.Context, account players.AccountID) (titles.ID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return "", s.failWith
	}
	return s.chosen[account], nil
}

func (s *Store) Choices(_ context.Context, accounts []players.AccountID) (map[players.AccountID]titles.ID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	choices := map[players.AccountID]titles.ID{}
	for _, account := range accounts {
		if choice, chosen := s.chosen[account]; chosen {
			choices[account] = choice
		}
	}
	return choices, nil
}

func (s *Store) Wear(_ context.Context, account players.AccountID, title titles.ID, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	s.chosen[account] = title
	return nil
}

func (s *Store) DeleteAccount(_ context.Context, account players.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	delete(s.chosen, account)
	return nil
}
