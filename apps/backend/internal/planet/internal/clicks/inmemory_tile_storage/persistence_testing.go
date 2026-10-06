//go:build testing

package inmemory_tile_storage

import (
	"context"
	"maps"
	"slices"
	"sync"
)

type MemoryPersistence struct {
	mu      sync.Mutex
	rows    map[uint32]string
	shields map[uint32]int
	saves   [][]Tile
	failing error
}

func NewMemoryPersistence(rows map[uint32]string) *MemoryPersistence {
	held := make(map[uint32]string, len(rows))
	maps.Copy(held, rows)

	return &MemoryPersistence{rows: held, shields: map[uint32]int{}}
}

func (m *MemoryPersistence) Shield(tile uint32, shields int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.shields[tile] = shields
}

func (m *MemoryPersistence) Load(_ context.Context, visit func(tile uint32, owner string, shields int)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for tile, owner := range m.rows {
		visit(tile, owner, m.shields[tile])
	}
	return nil
}

func (m *MemoryPersistence) Save(_ context.Context, tiles []Tile) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.saves = append(m.saves, slices.Clone(tiles))
	for _, tile := range tiles {
		if tile.Owner == "" {
			delete(m.rows, tile.ID)
			delete(m.shields, tile.ID)
			continue
		}
		m.rows[tile.ID] = tile.Owner
		if tile.Shields == 0 {
			delete(m.shields, tile.ID)
		} else {
			m.shields[tile.ID] = tile.Shields
		}
	}
	return nil
}

func (m *MemoryPersistence) Stored() map[uint32]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.rows)
}

func (m *MemoryPersistence) StoredShields() map[uint32]int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.shields)
}

func (m *MemoryPersistence) Saves() [][]Tile {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.saves)
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
