//go:build testing

package refusals

import (
	"context"
	"sync"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	rows    map[Caller]Snapshot
	failing error
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[Caller]Snapshot{}}
}

func (m *MemoryStore) Refusal(_ context.Context, caller Caller) (Refusal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Refusal{}, m.failing
	}
	return m.rows[caller].Restore(), nil
}

func (m *MemoryStore) Change(_ context.Context, caller Caller, change func(Refusal) Refusal) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	before := m.rows[caller]
	if kept := change(before.Restore()).TakeSnapshot(); kept != before {
		m.rows[caller] = kept
	}
	return nil
}

func (m *MemoryStore) Kept(caller Caller) Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.rows[caller]
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}
