//go:build testing

package cpflags

import (
	"context"
	"sync"
	"time"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	rows    map[string]HeardSnapshot
	failing error
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[string]HeardSnapshot{}}
}

func (m *MemoryStore) Heard(_ context.Context, flag string) (Heard, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failing != nil {
		return Heard{}, m.failing
	}
	return m.rows[flag].Restore()
}

func (m *MemoryStore) Change(_ context.Context, flag string, change func(Heard) Heard) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failing != nil {
		return m.failing
	}
	before := m.rows[flag]
	heard, err := before.Restore()
	if err != nil {
		return err
	}
	if after := change(heard).TakeSnapshot(); after != before {
		m.rows[flag] = after
	}
	return nil
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failing = err
}

// NewOpenCatalog is a catalog that heard each flag opened, and tells nobody.
func NewOpenCatalog(flags ...string) Catalog {
	store := NewMemoryStore()
	at := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	for _, flag := range flags {
		store.rows[flag] = HeardSnapshot{State: Open, DecidedAt: at}
	}
	return NewCatalog(store)
}
