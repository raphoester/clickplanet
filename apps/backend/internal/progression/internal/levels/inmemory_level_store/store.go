//go:build testing

package inmemory_level_store

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/levels"
)

type Store struct {
	mu   sync.Mutex
	kept levels.ByAccount
}

var _ levels.Store = (*Store)(nil)

func New() *Store {
	return &Store{kept: levels.ByAccount{}}
}

func (s *Store) Levels(_ context.Context, asked []accounts.AccountID) (levels.ByAccount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := levels.ByAccount{}
	for _, account := range asked {
		if level, held := s.kept[account]; held {
			found[account] = level
		}
	}
	return found, nil
}

func (s *Store) Raise(_ context.Context, account accounts.AccountID, level levels.Level, _ time.Time) (levels.Level, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	previous := s.kept[account]
	s.kept[account] = max(previous, level)
	return previous, nil
}

func (s *Store) RaiseAll(_ context.Context, raised levels.ByAccount, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for account, level := range raised {
		s.kept[account] = max(s.kept[account], level)
	}
	return nil
}

func (s *Store) Forget(_ context.Context, forgotten ...accounts.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, account := range forgotten {
		delete(s.kept, account)
	}
	return nil
}
