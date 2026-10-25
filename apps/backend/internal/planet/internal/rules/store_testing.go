//go:build testing

package rules

import (
	"context"
	"sync"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	kept    []Snapshot
	failing error
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) Change(_ context.Context, request RequestID) (Change, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Change{}, m.failing
	}
	for _, kept := range m.kept {
		if kept.Request == request {
			return kept.Restore(), nil
		}
	}
	return Change{}, nil
}

func (m *MemoryStore) Latest(context.Context) (Change, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Change{}, m.failing
	}
	var latest Snapshot
	for _, kept := range m.kept {
		if !kept.AppliedAt.Before(latest.AppliedAt) {
			latest = kept
		}
	}
	return latest.Restore(), nil
}

func (m *MemoryStore) Save(_ context.Context, change Change) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.kept = append(m.kept, change.TakeSnapshot())
	return nil
}

func (m *MemoryStore) Kept() []Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]Snapshot(nil), m.kept...)
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}
