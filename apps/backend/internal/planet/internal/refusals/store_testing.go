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
	reads   int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[Caller]Snapshot{}}
}

func (m *MemoryStore) Refusals(_ context.Context, callers []Caller) (map[Caller]Refusal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.reads++
	if m.failing != nil {
		return nil, m.failing
	}
	kept := map[Caller]Refusal{}
	for _, caller := range callers {
		if row, ok := m.rows[caller]; ok {
			kept[caller] = row.Restore()
		}
	}
	return kept, nil
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

func (m *MemoryStore) Reads() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.reads
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}
