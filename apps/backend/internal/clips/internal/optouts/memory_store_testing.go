//go:build testing

package optouts

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type MemoryStore struct {
	mu  sync.Mutex
	out *cpcolls.Set[AccountID]
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{out: cpcolls.NewSet[AccountID]()}
}

func (s *MemoryStore) Keep(_ context.Context, account AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.out.Add(account)
	return nil
}

func (s *MemoryStore) Forget(_ context.Context, account AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.out.Delete(account)
	return nil
}

func (s *MemoryStore) Among(_ context.Context, accounts []AccountID) ([]AccountID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []AccountID
	for _, account := range accounts {
		if s.out.Contains(account) {
			out = append(out, account)
		}
	}
	return out, nil
}
