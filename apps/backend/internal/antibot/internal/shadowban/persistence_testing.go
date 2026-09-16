//go:build testing

package shadowban

import (
	"context"
	"maps"
	"slices"
	"sync"
)

// MemoryPersistence is a Persistence held in a map, for tests that need a banner but not postgres.
type MemoryPersistence struct {
	mu      sync.Mutex
	rows    map[string]Record
	saves   [][]Record
	failing error
}

func NewMemoryPersistence(records ...Record) *MemoryPersistence {
	rows := make(map[string]Record, len(records))
	for _, record := range records {
		rows[record.Key] = record
	}

	return &MemoryPersistence{rows: rows}
}

func (m *MemoryPersistence) Load(_ context.Context, visit func(record Record)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for _, record := range m.rows {
		visit(record)
	}
	return nil
}

func (m *MemoryPersistence) Save(_ context.Context, records []Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.saves = append(m.saves, slices.Clone(records))
	for _, record := range records {
		m.rows[record.Key] = record
	}
	return nil
}

func (m *MemoryPersistence) Stored() map[string]Record {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.rows)
}

// Saves is every call to Save, as it was made.
func (m *MemoryPersistence) Saves() [][]Record {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.saves)
}

// FailWith makes every Load and Save return err until Heal.
func (m *MemoryPersistence) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}

func (m *MemoryPersistence) Heal() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = nil
}
