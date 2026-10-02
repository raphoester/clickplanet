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
	mu         sync.Mutex
	held       map[players.AccountID]titles.IDs
	backfilled titles.IDs
	failWith   error
}

var _ titles.Store = (*Store)(nil)

func New() *Store {
	return &Store{held: map[players.AccountID]titles.IDs{}}
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

func (s *Store) Grant(_ context.Context, grants titles.Grants, _ time.Time) error {
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

func (s *Store) DeleteAccount(_ context.Context, account players.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	delete(s.held, account)
	return nil
}

func (s *Store) Backfilled(context.Context) (titles.IDs, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return nil, s.failWith
	}
	return slices.Clone(s.backfilled), nil
}

func (s *Store) SaveBackfilled(_ context.Context, ids titles.IDs, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.failWith != nil {
		return s.failWith
	}
	s.backfilled = append(s.backfilled, ids.Without(s.backfilled)...)
	return nil
}
