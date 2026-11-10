//go:build testing

package warnings

import (
	"context"
	"slices"
	"sync"
	"time"
)

var _ Store = (*MemoryStore)(nil)

type MemoryStore struct {
	mu      sync.Mutex
	rows    map[string]Warning
	failing error
}

func NewMemoryStore(warnings ...Warning) *MemoryStore {
	rows := make(map[string]Warning, len(warnings))
	for _, warning := range warnings {
		rows[warning.Account] = warning
	}
	return &MemoryStore{rows: rows}
}

func (m *MemoryStore) Warned(_ context.Context, accounts []string) (map[string]Warning, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	warned := map[string]Warning{}
	for _, account := range accounts {
		if warning, ok := m.rows[account]; ok {
			warned[account] = warning
		}
	}
	return warned, nil
}

func (m *MemoryStore) Since(_ context.Context, from time.Time) ([]Warning, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	var since []Warning
	for _, warning := range m.rows {
		if !warning.At.Before(from) {
			since = append(since, warning)
		}
	}
	return since, nil
}

func (m *MemoryStore) Keep(_ context.Context, warning Warning) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return false, m.failing
	}
	if _, warned := m.rows[warning.Account]; warned {
		return false, nil
	}
	m.rows[warning.Account] = warning
	return true, nil
}

func (m *MemoryStore) Acknowledge(_ context.Context, account string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	warning, warned := m.rows[account]
	if !warned || !warning.Pending() {
		return nil
	}
	warning.AcknowledgedAt = at
	m.rows[account] = warning
	return nil
}

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}

var _ Feed = (*MemoryFeed)(nil)

type MemoryFeed struct {
	mu        sync.Mutex
	published []Warning
}

func (m *MemoryFeed) Publish(warning Warning) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.published = append(m.published, warning)
}

func (m *MemoryFeed) Published() []Warning {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.published)
}
