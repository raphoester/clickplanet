//go:build testing

package arenas

import (
	"context"
	"sync"
)

// MemoryStore keeps the arenas in a map.
type MemoryStore struct {
	mu      sync.Mutex
	arenas  map[ID]Arena
	Failing error
}

var _ Store = (*MemoryStore)(nil)

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{arenas: map[ID]Arena{}}
}

func (m *MemoryStore) Find(_ context.Context, id ID) (Arena, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return Arena{}, m.Failing
	}
	if arena, kept := m.arenas[id]; kept {
		return arena, nil
	}
	return Arena{ID: id}, nil
}

func (m *MemoryStore) Playing(context.Context) (Arena, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return Arena{}, m.Failing
	}
	for _, arena := range m.arenas {
		if arena.Opened && !arena.Closed {
			return arena, nil
		}
	}
	return Arena{}, nil
}

func (m *MemoryStore) Open(_ context.Context, opening Opening) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return m.Failing
	}
	m.arenas[opening.ID] = Arena{ID: opening.ID, Opening: opening, Opened: true}
	return nil
}

func (m *MemoryStore) Close(_ context.Context, closing Closing) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return m.Failing
	}
	arena := m.arenas[closing.ID]
	arena.ID, arena.Closed = closing.ID, true
	m.arenas[closing.ID] = arena
	return nil
}
