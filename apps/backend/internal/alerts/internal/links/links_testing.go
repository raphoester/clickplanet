//go:build testing

package links

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
)

type InMemoryStore struct {
	mu    sync.Mutex
	links map[AccountID]LinkSnapshot
}

var _ Store = (*InMemoryStore)(nil)

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{links: map[AccountID]LinkSnapshot{}}
}

func (s *InMemoryStore) LinkOf(_ context.Context, account AccountID) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept, found := s.links[account]
	if !found {
		return Link{}, ErrNoLink
	}
	return kept.Restore(), nil
}

func (s *InMemoryStore) LinkOfUser(_ context.Context, user discord.UserID) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, kept := range s.links {
		if kept.User == user {
			return kept.Restore(), nil
		}
	}
	return Link{}, ErrNoLink
}

func (s *InMemoryStore) SaveLinks(_ context.Context, saved ...Link) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, link := range saved {
		kept := link.TakeSnapshot()
		if kept.Forgotten {
			delete(s.links, kept.Account)
			continue
		}
		s.links[kept.Account] = kept
	}
	return nil
}

func (s *InMemoryStore) Links() []LinkSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := make([]LinkSnapshot, 0, len(s.links))
	for _, link := range s.links {
		kept = append(kept, link)
	}
	return kept
}
