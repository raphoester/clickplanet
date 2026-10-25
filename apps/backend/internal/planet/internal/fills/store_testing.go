//go:build testing

package fills

import (
	"context"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
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

func (m *MemoryStore) Fill(_ context.Context, request bonuses.FillID) (Fill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Fill{}, m.failing
	}
	for _, kept := range m.kept {
		if kept.ID == request {
			return kept.Restore()
		}
	}
	return Fill{}, nil
}

func (m *MemoryStore) Unfinished(context.Context) (Fill, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Fill{}, m.failing
	}
	var first *Snapshot
	for i, kept := range m.kept {
		if kept.FinishedAt.IsZero() && (first == nil || kept.StartedAt.Before(first.StartedAt)) {
			first = &m.kept[i]
		}
	}
	if first == nil {
		return Fill{}, nil
	}
	return first.Restore()
}

func (m *MemoryStore) Save(_ context.Context, fill Fill) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	kept := fill.TakeSnapshot()
	for i := range m.kept {
		if m.kept[i].ID == kept.ID {
			m.kept[i] = kept
			return nil
		}
	}
	m.kept = append(m.kept, kept)
	return nil
}

func (m *MemoryStore) Keep(kept Snapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.kept = append(m.kept, kept)
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
