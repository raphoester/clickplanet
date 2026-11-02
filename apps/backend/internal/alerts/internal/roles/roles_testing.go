//go:build testing

package roles

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/discord"
)

type InMemoryStore struct {
	mu    sync.Mutex
	roles map[Key]Kept
}

var _ Store = (*InMemoryStore)(nil)

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{roles: map[Key]Kept{}}
}

func (s *InMemoryStore) RoleOf(_ context.Context, key Key) (Kept, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept, found := s.roles[key]
	if !found {
		return Kept{}, ErrNoRole
	}
	return kept, nil
}

func (s *InMemoryStore) SaveRole(_ context.Context, key Key, kept Kept) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roles[key] = kept
	return nil
}

func (s *InMemoryStore) Keys(_ context.Context) ([]Key, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]Key, 0, len(s.roles))
	for key := range s.roles {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys, nil
}

func (s *InMemoryStore) ForgetRole(_ context.Context, key Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.roles, key)
	return nil
}

// Makes role "1", "2", … and remembers each name, each paint and each deletion. A role in Gone is one someone deleted
// by hand: painting it is discord.ErrNoRole.
type FakeMaker struct {
	mu      sync.Mutex
	Made    []string
	Painted map[discord.RoleID]discord.Color
	Deleted []discord.RoleID
	Gone    map[discord.RoleID]bool
}

func (m *FakeMaker) MakeRole(_ context.Context, name string, color discord.Color) (discord.RoleID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Made = append(m.Made, name)
	id := discord.RoleID(fmt.Sprint(len(m.Made)))
	m.paint(id, color)
	return id, nil
}

func (m *FakeMaker) PaintRole(_ context.Context, role discord.RoleID, color discord.Color) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Gone[role] {
		return discord.ErrNoRole
	}
	m.paint(role, color)
	return nil
}

func (m *FakeMaker) DeleteRole(_ context.Context, role discord.RoleID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Deleted = append(m.Deleted, role)
	return nil
}

func (m *FakeMaker) paint(role discord.RoleID, color discord.Color) {
	if m.Painted == nil {
		m.Painted = map[discord.RoleID]discord.Color{}
	}
	m.Painted[role] = color
}
