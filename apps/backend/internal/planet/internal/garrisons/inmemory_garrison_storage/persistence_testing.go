//go:build testing

package inmemory_garrison_storage

import (
	"context"
	"maps"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
)

type MemoryPersistence struct {
	mu        sync.Mutex
	garrisons map[uint32]garrisons.Garrison
	saves     int
	failing   error
}

func NewMemoryPersistence(stored ...garrisons.Garrison) *MemoryPersistence {
	m := &MemoryPersistence{garrisons: map[uint32]garrisons.Garrison{}}
	for _, garrison := range stored {
		m.garrisons[garrison.Tile] = garrison
	}

	return m
}

func (m *MemoryPersistence) Load(_ context.Context, visit func(garrisons.Garrison)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for _, garrison := range m.garrisons {
		visit(garrison)
	}

	return nil
}

func (m *MemoryPersistence) Save(_ context.Context, changed []garrisons.Garrison) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.saves++

	for _, garrison := range changed {
		if garrison.Empty() {
			delete(m.garrisons, garrison.Tile)
			continue
		}
		m.garrisons[garrison.Tile] = garrison
	}

	return nil
}

func (m *MemoryPersistence) Stored() map[uint32]garrisons.Garrison {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.garrisons)
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
