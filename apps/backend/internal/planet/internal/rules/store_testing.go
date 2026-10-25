//go:build testing

package rules

import (
	"context"
	"fmt"
	"sync"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	sets    map[SetName]Snapshot
	failing error
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{sets: map[SetName]Snapshot{}}
}

func (m *MemoryStore) Set(_ context.Context, name SetName) (Set, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Set{}, m.failing
	}
	kept, ok := m.sets[name]
	if !ok {
		return Set{}, fmt.Errorf("%q: %w", name, ErrUnknownSet)
	}
	return kept.Restore()
}

func (m *MemoryStore) Save(_ context.Context, set Set) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	kept := set.TakeSnapshot()
	m.sets[kept.Name] = kept
	return nil
}

func (m *MemoryStore) Kept(name SetName) (Snapshot, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	kept, ok := m.sets[name]
	return kept, ok
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}
