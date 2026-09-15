//go:build testing

package evidence

import (
	"context"
	"maps"
	"sync"
)

// MemoryPersistence is a Persistence held in memory, for tests that need a store but not postgres.
type MemoryPersistence struct {
	mu       sync.Mutex
	snapshot Snapshot
	saves    int
	failing  error
}

func NewMemoryPersistence() *MemoryPersistence {
	return &MemoryPersistence{}
}

func (m *MemoryPersistence) Load(context.Context) (Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Snapshot{}, m.failing
	}
	return Snapshot{SavedAt: m.snapshot.SavedAt, Sections: maps.Clone(m.snapshot.Sections)}, nil
}

func (m *MemoryPersistence) Save(_ context.Context, snapshot Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.saves++
	m.snapshot = Snapshot{SavedAt: snapshot.SavedAt, Sections: maps.Clone(snapshot.Sections)}
	return nil
}

// Saves is how many times Save succeeded.
func (m *MemoryPersistence) Saves() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.saves
}

// FailWith makes every Load and Save return err until Heal.
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
