//go:build testing

package resets

import (
	"context"
	"slices"
	"sync"
	"time"
)

// MemoryStore keeps the resets in a slice.
type MemoryStore struct {
	mu      sync.Mutex
	kept    []Reset
	Failing error
}

var _ Store = (*MemoryStore)(nil)

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) Keep(_ context.Context, reset Reset) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return false, m.Failing
	}
	if slices.ContainsFunc(m.kept, func(kept Reset) bool { return kept.ID == reset.ID }) {
		return false, nil
	}
	m.kept = append(m.kept, reset)
	return true, nil
}

func (m *MemoryStore) After(_ context.Context, at time.Time) ([]Reset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Failing != nil {
		return nil, m.Failing
	}
	var after []Reset
	for _, kept := range m.kept {
		if kept.At.After(at) {
			after = append(after, kept)
		}
	}
	slices.SortFunc(after, func(a, b Reset) int { return a.At.Compare(b.At) })
	return after, nil
}
