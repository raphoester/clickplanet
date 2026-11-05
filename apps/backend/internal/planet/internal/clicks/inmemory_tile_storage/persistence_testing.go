//go:build testing

package inmemory_tile_storage

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type MemoryPersistence struct {
	mu         sync.Mutex
	rows       map[uint32]string
	shields    map[uint32]int
	holders    map[uint32]cpsession.AccountID
	landmasses map[string]map[clicks.LandmassID]string
	saves      [][]Tile
	failing    error
}

func NewMemoryPersistence(rows map[uint32]string) *MemoryPersistence {
	held := make(map[uint32]string, len(rows))
	maps.Copy(held, rows)

	return &MemoryPersistence{
		rows: held, shields: map[uint32]int{}, holders: map[uint32]cpsession.AccountID{},
		landmasses: map[string]map[clicks.LandmassID]string{},
	}
}

func (m *MemoryPersistence) Fortified(asset string, landmass clicks.LandmassID, flag string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.landmasses[asset] == nil {
		m.landmasses[asset] = map[clicks.LandmassID]string{}
	}
	m.landmasses[asset][landmass] = flag
}

func (m *MemoryPersistence) LoadLandmasses(
	_ context.Context, asset string, visit func(landmass clicks.LandmassID, fortifiedBy string),
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for landmass, flag := range m.landmasses[asset] {
		visit(landmass, flag)
	}
	return nil
}

func (m *MemoryPersistence) StoredLandmasses(asset string) map[clicks.LandmassID]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.landmasses[asset])
}

func (m *MemoryPersistence) Shield(tile uint32, shields int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.shields[tile] = shields
}

func (m *MemoryPersistence) Hold(tile uint32, account cpsession.AccountID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.holders[tile] = account
}

func (m *MemoryPersistence) Load(_ context.Context, visit func(tile Tile)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for tile, owner := range m.rows {
		visit(Tile{ID: tile, Owner: owner, Shields: m.shields[tile], Holder: m.holders[tile]})
	}
	return nil
}

func (m *MemoryPersistence) Save(_ context.Context, tiles []Tile, landmasses []Landmass) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for _, landmass := range landmasses {
		if m.landmasses[landmass.Asset] == nil {
			m.landmasses[landmass.Asset] = map[clicks.LandmassID]string{}
		}
		m.landmasses[landmass.Asset][landmass.ID] = landmass.FortifiedBy
	}
	if len(tiles) == 0 {
		return nil
	}
	m.saves = append(m.saves, slices.Clone(tiles))
	for _, tile := range tiles {
		if tile.Owner == "" {
			delete(m.rows, tile.ID)
			delete(m.shields, tile.ID)
			delete(m.holders, tile.ID)
			continue
		}
		m.rows[tile.ID] = tile.Owner
		if tile.Holder == cpsession.NoAccount {
			delete(m.holders, tile.ID)
		} else {
			m.holders[tile.ID] = tile.Holder
		}
		if tile.Shields == 0 {
			delete(m.shields, tile.ID)
		} else {
			m.shields[tile.ID] = tile.Shields
		}
	}
	return nil
}

func (m *MemoryPersistence) Wipe(ctx context.Context, keep func(ctx context.Context) (bool, error)) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return false, m.failing
	}
	kept, err := keep(ctx)
	if err != nil || !kept {
		return false, err
	}
	m.rows, m.shields, m.holders = map[uint32]string{}, map[uint32]int{}, map[uint32]cpsession.AccountID{}
	m.landmasses = map[string]map[clicks.LandmassID]string{}
	return true, nil
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

func (m *MemoryPersistence) StoredHolders() map[uint32]cpsession.AccountID {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.holders)
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
