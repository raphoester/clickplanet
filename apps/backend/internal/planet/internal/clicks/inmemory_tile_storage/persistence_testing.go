//go:build testing

package inmemory_tile_storage

import (
	"context"
	"maps"
	"slices"
	"sync"
)

// MemoryPersistence is a Persistence held in a map, for tests that need a storage but not postgres.
type MemoryPersistence struct {
	mu      sync.Mutex
	rows    map[uint32]string
	saves   []Save
	failing error
}

// Save is one call to MemoryPersistence.Save, as it was made.
type Save struct {
	Tiles  []uint32
	Owners []string
}

// NewMemoryPersistence starts holding rows, as a table would.
func NewMemoryPersistence(rows map[uint32]string) *MemoryPersistence {
	held := make(map[uint32]string, len(rows))
	maps.Copy(held, rows)

	return &MemoryPersistence{rows: held}
}

func (m *MemoryPersistence) Load(_ context.Context, visit func(tile uint32, owner string)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for tile, owner := range m.rows {
		visit(tile, owner)
	}
	return nil
}

func (m *MemoryPersistence) Save(_ context.Context, tiles []uint32, owners []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.saves = append(m.saves, Save{Tiles: slices.Clone(tiles), Owners: slices.Clone(owners)})
	for i, tile := range tiles {
		if owners[i] == "" {
			delete(m.rows, tile)
			continue
		}
		m.rows[tile] = owners[i]
	}
	return nil
}

func (m *MemoryPersistence) Stored() map[uint32]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.rows)
}

func (m *MemoryPersistence) Saves() []Save {
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
