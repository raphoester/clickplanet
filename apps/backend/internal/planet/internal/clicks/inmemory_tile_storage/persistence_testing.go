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
	mu      sync.Mutex
	maps    map[clicks.MapID]*memoryMap
	saves   [][]Tile
	failing error
}

type memoryMap struct {
	rows       map[uint32]string
	shields    map[uint32]int
	holders    map[uint32]cpsession.AccountID
	landmasses map[string]map[clicks.LandmassID]string
	out        map[string]OutFlag
}

func newMemoryMap() *memoryMap {
	return &memoryMap{
		rows: map[uint32]string{}, shields: map[uint32]int{}, holders: map[uint32]cpsession.AccountID{},
		landmasses: map[string]map[clicks.LandmassID]string{}, out: map[string]OutFlag{},
	}
}

// NewMemoryPersistence keeps rows as the world's map.
func NewMemoryPersistence(rows map[uint32]string) *MemoryPersistence {
	world := newMemoryMap()
	maps.Copy(world.rows, rows)
	return &MemoryPersistence{maps: map[clicks.MapID]*memoryMap{clicks.World: world}}
}

func (m *MemoryPersistence) mapLocked(played clicks.MapID) *memoryMap {
	if m.maps[played] == nil {
		m.maps[played] = newMemoryMap()
	}
	return m.maps[played]
}

func (m *MemoryPersistence) Fortified(asset string, landmass clicks.LandmassID, flag string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	world := m.mapLocked(clicks.World)
	if world.landmasses[asset] == nil {
		world.landmasses[asset] = map[clicks.LandmassID]string{}
	}
	world.landmasses[asset][landmass] = flag
}

func (m *MemoryPersistence) LoadLandmasses(
	_ context.Context, played clicks.MapID, asset string, visit func(landmass clicks.LandmassID, fortifiedBy string),
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for landmass, flag := range m.mapLocked(played).landmasses[asset] {
		visit(landmass, flag)
	}
	return nil
}

// StoredLandmasses is the world's.
func (m *MemoryPersistence) StoredLandmasses(asset string) map[clicks.LandmassID]string {
	return m.StoredLandmassesOn(clicks.World, asset)
}

func (m *MemoryPersistence) StoredLandmassesOn(played clicks.MapID, asset string) map[clicks.LandmassID]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.mapLocked(played).landmasses[asset])
}

func (m *MemoryPersistence) Shield(tile uint32, shields int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.mapLocked(clicks.World).shields[tile] = shields
}

func (m *MemoryPersistence) Hold(tile uint32, account cpsession.AccountID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.mapLocked(clicks.World).holders[tile] = account
}

func (m *MemoryPersistence) Load(_ context.Context, played clicks.MapID, visit func(tile Tile)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	kept := m.mapLocked(played)
	for tile, owner := range kept.rows {
		visit(Tile{ID: tile, Owner: owner, Shields: kept.shields[tile], Holder: kept.holders[tile]})
	}
	return nil
}

func (m *MemoryPersistence) LoadOutFlags(_ context.Context, played clicks.MapID, visit func(out OutFlag)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for _, out := range m.mapLocked(played).out {
		visit(out)
	}
	return nil
}

// StoredOutFlagsOn is a map's flags out, by flag.
func (m *MemoryPersistence) StoredOutFlagsOn(played clicks.MapID) map[string]OutFlag {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.mapLocked(played).out)
}

func (m *MemoryPersistence) Save(
	_ context.Context, played clicks.MapID, tiles []Tile, landmasses []Landmass, out []OutFlag,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	if len(tiles) > 0 {
		m.saves = append(m.saves, slices.Clone(tiles))
	}
	kept := m.mapLocked(played)
	for _, flag := range out {
		if _, ok := kept.out[flag.Flag]; !ok {
			kept.out[flag.Flag] = flag
		}
	}
	for _, landmass := range landmasses {
		if kept.landmasses[landmass.Asset] == nil {
			kept.landmasses[landmass.Asset] = map[clicks.LandmassID]string{}
		}
		kept.landmasses[landmass.Asset][landmass.ID] = landmass.FortifiedBy
	}
	for _, tile := range tiles {
		if tile.Owner == "" {
			delete(kept.rows, tile.ID)
			delete(kept.shields, tile.ID)
			delete(kept.holders, tile.ID)
			continue
		}
		kept.rows[tile.ID] = tile.Owner
		if tile.Holder == cpsession.NoAccount {
			delete(kept.holders, tile.ID)
		} else {
			kept.holders[tile.ID] = tile.Holder
		}
		if tile.Shields == 0 {
			delete(kept.shields, tile.ID)
		} else {
			kept.shields[tile.ID] = tile.Shields
		}
	}
	return nil
}

func (m *MemoryPersistence) Open(
	ctx context.Context, played clicks.MapID, tiles []Tile, keep func(ctx context.Context) (bool, error),
) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return false, m.failing
	}
	kept, err := keep(ctx)
	if err != nil || !kept {
		return false, err
	}
	opened := m.mapLocked(played)
	for _, tile := range tiles {
		opened.rows[tile.ID] = tile.Owner
		opened.shields[tile.ID] = tile.Shields
	}
	return true, nil
}

// StoredShieldsOn is a map's shields.
func (m *MemoryPersistence) StoredShieldsOn(played clicks.MapID) map[uint32]int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.mapLocked(played).shields)
}

func (m *MemoryPersistence) Wipe(ctx context.Context, played clicks.MapID, keep func(ctx context.Context) (bool, error)) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return false, m.failing
	}
	kept, err := keep(ctx)
	if err != nil || !kept {
		return false, err
	}
	m.maps[played] = newMemoryMap()
	return true, nil
}

func (m *MemoryPersistence) ForgetHolder(_ context.Context, account cpsession.AccountID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for _, kept := range m.maps {
		for tile, holder := range kept.holders {
			if holder == account {
				delete(kept.holders, tile)
			}
		}
	}
	return nil
}

// Stored is the world's.
func (m *MemoryPersistence) Stored() map[uint32]string {
	return m.StoredOn(clicks.World)
}

func (m *MemoryPersistence) StoredOn(played clicks.MapID) map[uint32]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.mapLocked(played).rows)
}

func (m *MemoryPersistence) StoredShields() map[uint32]int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.mapLocked(clicks.World).shields)
}

func (m *MemoryPersistence) StoredHolders() map[uint32]cpsession.AccountID {
	return m.StoredHoldersOn(clicks.World)
}

func (m *MemoryPersistence) StoredHoldersOn(played clicks.MapID) map[uint32]cpsession.AccountID {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.mapLocked(played).holders)
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
