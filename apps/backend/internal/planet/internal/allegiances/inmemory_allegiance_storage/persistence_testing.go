//go:build testing

package inmemory_allegiance_storage

import (
	"context"
	"maps"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
)

type MemoryPersistence struct {
	mu      sync.Mutex
	kept    map[allegiances.Account]allegiances.Snapshot
	failing error
}

func NewMemoryPersistence() *MemoryPersistence {
	return &MemoryPersistence{kept: map[allegiances.Account]allegiances.Snapshot{}}
}

func (m *MemoryPersistence) Load(_ context.Context, visit func(allegiances.Account, allegiances.Snapshot)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for account, kept := range m.kept {
		visit(account, kept)
	}
	return nil
}

func (m *MemoryPersistence) Save(_ context.Context, kept map[allegiances.Account]allegiances.Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	maps.Copy(m.kept, kept)
	return nil
}

func (m *MemoryPersistence) Stored() map[allegiances.Account]allegiances.Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.kept)
}

func (m *MemoryPersistence) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}

func (m *MemoryPersistence) Heal() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = nil
}
