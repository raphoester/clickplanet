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

func (m *MemoryStore) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.failing = err
}

var _ Teller = (*MemoryTeller)(nil)

type MemoryTeller struct {
	mu   sync.Mutex
	told []Warning
}

func (m *MemoryTeller) Tell(_ context.Context, warning Warning) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.told = append(m.told, warning)
	return nil
}

func (m *MemoryTeller) Told() []Warning {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.told)
}
