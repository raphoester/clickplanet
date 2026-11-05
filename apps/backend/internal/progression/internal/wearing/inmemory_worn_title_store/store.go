//go:build testing

package inmemory_worn_title_store

import (
	"context"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/titles"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/wearing"
)

type Store struct {
	mu       sync.Mutex
	chosen   map[accounts.AccountID]titles.ID
	failWith error
}

var _ wearing.Store = (*Store)(nil)

func New() *Store {
	return &Store{chosen: map[accounts.AccountID]titles.ID{}}
}

func (s *Store) FailWith(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.failWith = err
}

func (s *Store) Choice(_ context.Context, account accounts.AccountID) (titles.ID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return "", s.failWith
	}
	return s.chosen[account], nil
}

func (s *Store) Wear(_ context.Context, account accounts.AccountID, title titles.ID, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	s.chosen[account] = title
	return nil
}

func (s *Store) DeleteAccount(_ context.Context, account accounts.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	delete(s.chosen, account)
	return nil
}
