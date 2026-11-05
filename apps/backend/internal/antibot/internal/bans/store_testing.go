//go:build testing

package bans

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	rows    map[string]Record
	failing error
}

func NewMemoryStore(records ...Record) *MemoryStore {
	rows := make(map[string]Record, len(records))
	for _, record := range records {
		rows[record.Key] = record
	}

	return &MemoryStore{rows: rows}
}

func (m *MemoryStore) Record(_ context.Context, key string) (Record, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Record{}, false, m.failing
	}
	record, found := m.rows[key]
	return record, found, nil
}

func (m *MemoryStore) Running(_ context.Context, now time.Time) ([]Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	var running []Record
	for _, record := range m.rows {
		if record.running(now) {
			running = append(running, record)
		}
	}
	return running, nil
}

func (m *MemoryStore) Change(_ context.Context, key string, change func(Record, bool) (Record, bool)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	record, found := m.rows[key]
	if !found {
		record = Record{Key: key}
	}
	if next, keep := change(record, found); keep {
		next.Key = key
		m.rows[key] = next
	}
	return nil
}

func (m *MemoryStore) Stored() map[string]Record {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.rows)
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}

var _ Enforcer = (*MemoryEnforcer)(nil)

type Told struct {
	Caller Caller
	Until  time.Time
	At     time.Time
}

type MemoryEnforcer struct {
	mu   sync.Mutex
	told []Told
}

func (m *MemoryEnforcer) Enforce(_ context.Context, caller Caller, until, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.told = append(m.told, Told{Caller: caller, Until: until, At: at})
	return nil
}

func (m *MemoryEnforcer) Told() []Told {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.told)
}
