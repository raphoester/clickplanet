//go:build testing

package appearances

import (
	"context"
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
)

type MemoryStore struct {
	mu   sync.Mutex
	kept map[Key]Appearance
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{kept: map[Key]Appearance{}}
}

func (s *MemoryStore) Keep(_ context.Context, named []Appearance) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, appearance := range named {
		s.kept[Key{Post: appearance.Post.ID, Account: appearance.Account}] = appearance
	}
	return nil
}

func (s *MemoryStore) ScheduledBefore(_ context.Context, at time.Time) ([]Key, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var keys []Key
	for _, key := range s.sorted() {
		if s.kept[key].Post.ScheduledAt.Before(at) {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (s *MemoryStore) Forget(_ context.Context, key Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.kept, key)
	return nil
}

func (s *MemoryStore) ForgetAccount(_ context.Context, account casts.AccountID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key := range s.kept {
		if key.Account == account {
			delete(s.kept, key)
		}
	}
	return nil
}

// By post, then by account.
func (s *MemoryStore) Kept() []Appearance {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := make([]Appearance, 0, len(s.kept))
	for _, key := range s.sorted() {
		kept = append(kept, s.kept[key])
	}
	return kept
}

func (s *MemoryStore) sorted() []Key {
	return slices.SortedFunc(maps.Keys(s.kept), func(a, b Key) int {
		if order := strings.Compare(string(a.Post), string(b.Post)); order != 0 {
			return order
		}
		return strings.Compare(a.Account.String(), b.Account.String())
	})
}
