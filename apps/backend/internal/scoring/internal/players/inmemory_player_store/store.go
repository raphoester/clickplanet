//go:build testing

package inmemory_player_store

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Store struct {
	mu      sync.Mutex
	players *cpcolls.Set[players.AccountID]
}

var _ players.Store = (*Store)(nil)

func New() *Store {
	return &Store{players: cpcolls.NewSet[players.AccountID]()}
}

func (s *Store) Keep(_ context.Context, account players.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.players.Add(account)
	return nil
}

func (s *Store) IsPlayer(_ context.Context, account players.AccountID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.players.Contains(account), nil
}

func (s *Store) Forget(_ context.Context, account players.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.players.Delete(account)
	return nil
}
