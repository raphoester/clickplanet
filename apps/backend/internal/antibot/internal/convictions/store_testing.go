//go:build testing

package convictions

import (
	"context"
	"slices"
	"sync"
	"time"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	rows    []Conviction
	failing error
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) Keep(_ context.Context, conviction Conviction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	conviction.Readings = slices.Clone(conviction.Readings)
	m.rows = append(m.rows, conviction)
	return nil
}

func (m *MemoryStore) Between(_ context.Context, since, until time.Time) ([]Conviction, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	var between []Conviction
	for _, conviction := range m.rows {
		if !conviction.At.Before(since) && conviction.At.Before(until) {
			between = append(between, conviction)
		}
	}
	slices.SortStableFunc(between, func(a, b Conviction) int { return a.At.Compare(b.At) })
	return between, nil
}

func (m *MemoryStore) Stored() []Conviction {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.rows)
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}
