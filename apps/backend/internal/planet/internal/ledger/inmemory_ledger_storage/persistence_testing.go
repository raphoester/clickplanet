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
	events    map[ledger.Position]Stored
	head      ledger.Position
	forgotten map[ledger.Caller]ledger.Position
	saves     int
	failing   error
}

func NewMemoryPersistence() *MemoryPersistence {
	return &MemoryPersistence{
		events:    map[ledger.Position]Stored{},
		forgotten: map[ledger.Caller]ledger.Position{},
	}
}

func (m *MemoryPersistence) Load(_ context.Context, visit func(Stored)) (Marks, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return Marks{}, m.failing
	}
	for _, position := range slices.Sorted(maps.Keys(m.events)) {
		if position >= m.head {
			visit(m.events[position])
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

	maps.DeleteFunc(m.events, func(position ledger.Position, _ Stored) bool {
		return position >= changes.From
	})
	for entry := range changes.Events {
		m.events[entry.Position] = entry
	}
	for position, entry := range m.events {
		if position < changes.Marks.Head {
			m.events[position] = edited(entry, func(scope, _ *string) { *scope = "" })
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
	for position, entry := range m.events {
		m.events[position] = edited(entry, func(_, owner *string) {
			if *owner == account.String() {
				*owner = ""
			}
		})
	}
	return nil
}

func edited(entry Stored, edit func(scope, account *string)) Stored {
	if entry.Bombing == nil {
		edit(&entry.Taking.Scope, &entry.Taking.Account)
		return entry
	}

	bombing := *entry.Bombing
	edit(&bombing.Scope, &bombing.Account)
	entry.Bombing = &bombing
	return entry
}

func (m *MemoryPersistence) Stored() []Stored {
	m.mu.Lock()
	defer m.mu.Unlock()

	stored := make([]Stored, 0, len(m.events))
	for _, position := range slices.Sorted(maps.Keys(m.events)) {
		stored = append(stored, m.events[position])
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
