//go:build testing

package roles

import (
	"context"
	"fmt"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
)

type InMemoryStore struct {
	mu    sync.Mutex
	roles map[Key]discord.RoleID
}

var _ Store = (*InMemoryStore)(nil)

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{roles: map[Key]discord.RoleID{}}
}

func (s *InMemoryStore) RoleOf(_ context.Context, key Key) (discord.RoleID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, found := s.roles[key]
	if !found {
		return "", ErrNoRole
	}
	return id, nil
}

func (s *InMemoryStore) SaveRole(_ context.Context, key Key, id discord.RoleID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roles[key] = id
	return nil
}

// Makes role "1", "2", … and remembers each name.
type FakeMaker struct {
	mu   sync.Mutex
	Made []string
}

func (m *FakeMaker) MakeRole(_ context.Context, name string) (discord.RoleID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Made = append(m.Made, name)
	return discord.RoleID(fmt.Sprint(len(m.Made))), nil
}
