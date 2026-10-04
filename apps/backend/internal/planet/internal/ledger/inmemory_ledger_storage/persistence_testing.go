//go:build testing

package inmemory_ledger_storage

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

type MemoryPersistence struct {
	mu        sync.Mutex
	takes     map[ledger.Position]ledger.Taking
	head      ledger.Position
	forgotten map[ledger.Caller]ledger.Position
	saves     int
	failing   error
}

func NewMemoryPersistence() *MemoryPersistence {
	return &MemoryPersistence{
		takes:     map[ledger.Position]ledger.Taking{},
		forgotten: map[ledger.Caller]ledger.Position{},
	}
}

func (m *MemoryPersistence) Load(_ context.Context, visit func(Stored)) (Marks, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Marks{}, m.failing
	}
	for _, position := range slices.Sorted(maps.Keys(m.takes)) {
		if position >= m.head {
			visit(Stored{Position: position, Taking: m.takes[position]})
		}
	}
	return Marks{Head: m.head, Forgotten: maps.Clone(m.forgotten)}, nil
}

func (m *MemoryPersistence) Save(_ context.Context, changes Changes) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.saves++

	maps.DeleteFunc(m.takes, func(position ledger.Position, _ ledger.Taking) bool {
		return position >= changes.From
	})
	for take := range changes.Takes {
		m.takes[take.Position] = take.Taking
	}
	for position, taking := range m.takes {
		if position < changes.Marks.Head {
			taking.Scope = ""
			m.takes[position] = taking
		}
	}

	m.head = changes.Marks.Head
	maps.DeleteFunc(m.forgotten, func(_ ledger.Caller, before ledger.Position) bool { return before <= m.head })
	maps.Copy(m.forgotten, changes.Marks.Forgotten)
	return nil
}

func (m *MemoryPersistence) AnonymizeTakes(_ context.Context, account ledger.AccountID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for position, taking := range m.takes {
		if taking.Account == account.String() {
			taking.Account = ""
			m.takes[position] = taking
		}
	}
	return nil
}

func (m *MemoryPersistence) Stored() []Stored {
	m.mu.Lock()
	defer m.mu.Unlock()

	stored := make([]Stored, 0, len(m.takes))
	for _, position := range slices.Sorted(maps.Keys(m.takes)) {
		stored = append(stored, Stored{Position: position, Taking: m.takes[position]})
	}
	return stored
}

func (m *MemoryPersistence) Marks() Marks {
	m.mu.Lock()
	defer m.mu.Unlock()

	return Marks{Head: m.head, Forgotten: maps.Clone(m.forgotten)}
}

func (m *MemoryPersistence) Saves() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.saves
}

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
