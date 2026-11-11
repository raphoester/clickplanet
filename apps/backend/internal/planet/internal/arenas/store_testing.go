//go:build testing

package arenas

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"
)

// MemoryStore keeps the arenas in a map.
type MemoryStore struct {
	mu      sync.Mutex
	arenas  map[ID]Arena
	closed  map[ID]time.Time
	Failing error
}

var _ Store = (*MemoryStore)(nil)

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{arenas: map[ID]Arena{}, closed: map[ID]time.Time{}}
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

func (m *MemoryStore) Stretches(context.Context) ([]Stretch, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return nil, m.Failing
	}
	var stretches []Stretch
	for id, arena := range m.arenas {
		if arena.Opened {
			closedAt, closed := m.closed[id]
			stretches = append(stretches, Stretch{ID: id, OpenedAt: arena.Opening.OpenedAt, ClosedAt: closedAt, Closed: closed})
		}
	}
	slices.SortFunc(stretches, func(a, b Stretch) int { return cmp.Compare(a.OpenedAt.UnixNano(), b.OpenedAt.UnixNano()) })
	return stretches, nil
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
	arena.ID, arena.Closed, arena.ClosedAt = closing.ID, true, closing.ClosedAt.UTC()
	m.arenas[closing.ID] = arena
	m.closed[closing.ID] = closing.ClosedAt
	return nil
}
