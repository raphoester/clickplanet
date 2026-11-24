//go:build testing

package grants

import (
	"context"
	"sync"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu   sync.Mutex
	rows map[GrantID]Snapshot
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: map[GrantID]Snapshot{}}
}

func (m *MemoryStore) Grant(_ context.Context, id GrantID) (Grant, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	row, ok := m.rows[id]
	if !ok {
		return Unheard(id), nil
	}
	return row.Restore()
}

// Save keeps what postgres keeps: a grant once, and its taking back once.
func (m *MemoryStore) Save(_ context.Context, grant Grant) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	row := grant.TakeSnapshot()
	kept := m.rows[row.ID]
	kept.ID = row.ID
	if row.Granted && !kept.Granted {
		kept.Granted, kept.Flag, kept.Added, kept.GrantedAt = true, row.Flag, row.Added, row.GrantedAt
	}
	if row.TakenBack && !kept.TakenBack {
		kept.TakenBack, kept.TakenBackAt = true, row.TakenBackAt
	}
	m.rows[row.ID] = kept
	return nil
}
