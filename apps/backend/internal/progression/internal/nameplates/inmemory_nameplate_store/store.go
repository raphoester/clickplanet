//go:build testing

package inmemory_nameplate_store

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/nameplates"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type Store struct {
	mu    sync.Mutex
	named *cpcolls.Set[accounts.AccountID]
}

var _ nameplates.Store = (*Store)(nil)

func New() *Store {
	return &Store{named: cpcolls.NewSet[accounts.AccountID]()}
}

func (s *Store) Keep(_ context.Context, account accounts.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.named.Add(account)
	return nil
}

func (s *Store) Named(_ context.Context, account accounts.AccountID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.named.Contains(account), nil
}

func (s *Store) Forget(_ context.Context, account accounts.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.named.Delete(account)
	return nil
}
