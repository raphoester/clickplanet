//go:build testing

package allegiances

import (
	"context"
	"sync"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	rows    map[Account]Snapshot
	failing error
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[Account]Snapshot{}}
}

func (m *MemoryStore) Change(_ context.Context, account Account, change func(Allegiance) (Allegiance, error)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	before := m.rows[account]
	after, err := change(before.Restore())
	if err != nil {
		return err
	}
	if kept := after.TakeSnapshot(); kept != before {
		m.rows[account] = kept
	}
	return nil
}

func (m *MemoryStore) Kept(account Account) Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.rows[account]
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}
