//go:build testing

package arenas

import (
	"context"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// MemoryStore keeps the arenas in maps.
type MemoryStore struct {
	mu      sync.Mutex
	arenas  map[ID]Arena
	before  map[ID]clicks.MapCopy
	Failing error
}

var _ Store = (*MemoryStore)(nil)

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{arenas: map[ID]Arena{}, before: map[ID]clicks.MapCopy{}}
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

func (m *MemoryStore) Open(_ context.Context, opening Opening, before clicks.MapCopy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return m.Failing
	}
	m.arenas[opening.ID] = Arena{ID: opening.ID, Opening: opening, Opened: true}
	m.before[opening.ID] = clicks.MapCopy{Tiles: slices.Clone(before.Tiles), Landmasses: slices.Clone(before.Landmasses)}
	return nil
}

func (m *MemoryStore) Close(_ context.Context, closing Closing) (clicks.MapCopy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return clicks.MapCopy{}, m.Failing
	}
	arena := m.arenas[closing.ID]
	arena.ID, arena.Closed = closing.ID, true
	m.arenas[closing.ID] = arena
	before := m.before[closing.ID]
	delete(m.before, closing.ID)
	return before, nil
}

func (m *MemoryStore) ForgetHolder(_ context.Context, account cpsession.AccountID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return m.Failing
	}
	for _, before := range m.before {
		for i := range before.Tiles {
			if before.Tiles[i].Holder == account {
				before.Tiles[i].Holder = cpsession.NoAccount
			}
		}
	}
	return nil
}
