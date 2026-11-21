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
	// The board every row names: the last a write named, TestBoard for the rows a test seeds.
	board     clicks.BoardID
	rows      map[uint32]string
	shields   map[uint32]int
	holders   map[uint32]cpsession.AccountID
	provinces map[string]map[clicks.ProvinceID]string
	out       map[string]clicks.OutFlag
	viruses   []clicks.VirusSnapshot
	mines     map[uint32]clicks.MineSnapshot
}

func newMemoryMap() *memoryMap {
	return &memoryMap{
		board: clicks.TestBoard,
		rows:  map[uint32]string{}, shields: map[uint32]int{}, holders: map[uint32]cpsession.AccountID{},
		provinces: map[string]map[clicks.ProvinceID]string{}, out: map[string]clicks.OutFlag{},
		mines: map[uint32]clicks.MineSnapshot{},
	}
}

func (m *MemoryPersistence) LoadMines(_ context.Context, played clicks.MapID, visit func(mine clicks.MineSnapshot)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for _, tile := range slices.Sorted(maps.Keys(m.mapLocked(played).mines)) {
		visit(m.mapLocked(played).mines[tile])
	}
	return nil
}

func (m *MemoryPersistence) SaveMines(_ context.Context, played clicks.Stage, laid []clicks.MineSnapshot, gone []uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	kept := m.writeLocked(played).mines
	for _, mine := range laid {
		kept[mine.Tile] = mine
	}
	for _, tile := range gone {
		delete(kept, tile)
	}
	return nil
}

// StoredMinesOn is a map's mines, in tile order, as the last flush kept them.
func (m *MemoryPersistence) StoredMinesOn(played clicks.MapID) []clicks.MineSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	kept := m.mapLocked(played).mines
	mines := make([]clicks.MineSnapshot, 0, len(kept))
	for _, tile := range slices.Sorted(maps.Keys(kept)) {
		mines = append(mines, kept[tile])
	}
	return mines
}

func (m *MemoryPersistence) LoadViruses(_ context.Context, played clicks.MapID, visit func(virus clicks.VirusSnapshot)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for _, virus := range m.mapLocked(played).viruses {
		visit(virus)
	}
	return nil
}

func (m *MemoryPersistence) SaveViruses(_ context.Context, played clicks.Stage, live []clicks.VirusSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	m.writeLocked(played).viruses = slices.Clone(live)
	return nil
}

// StoredVirusesOn is a map's live viruses, as the last flush kept them.
func (m *MemoryPersistence) StoredVirusesOn(played clicks.MapID) []clicks.VirusSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	return slices.Clone(m.mapLocked(played).viruses)
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

// writeLocked is the map a write names the board of: every row it keeps names that board.
func (m *MemoryPersistence) writeLocked(played clicks.Stage) *memoryMap {
	kept := m.mapLocked(played.Map)
	kept.board = played.Board
	return kept
}

// Boards names the map's board when it keeps any row.
func (m *MemoryPersistence) Boards(_ context.Context, played clicks.MapID) ([]clicks.BoardID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return nil, m.failing
	}
	kept := m.mapLocked(played)
	if len(kept.rows) == 0 && len(kept.provinces) == 0 && len(kept.mines) == 0 && len(kept.viruses) == 0 {
		return nil, nil
	}
	return []clicks.BoardID{kept.board}, nil
}

// Remapped names another board on every row of a map, as rows written before a switch of boards the code never got.
func (m *MemoryPersistence) Remapped(played clicks.MapID, board clicks.BoardID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.mapLocked(played).board = board
}

// BoardOn is the board a map's rows name.
func (m *MemoryPersistence) BoardOn(played clicks.MapID) clicks.BoardID {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.mapLocked(played).board
}

func (m *MemoryPersistence) Fortified(asset string, province clicks.ProvinceID, flag string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	world := m.mapLocked(clicks.World)
	if world.provinces[asset] == nil {
		world.provinces[asset] = map[clicks.ProvinceID]string{}
	}
	world.provinces[asset][province] = flag
}

func (m *MemoryPersistence) LoadProvinces(
	_ context.Context, played clicks.MapID, asset string, visit func(province clicks.ProvinceID, fortifiedBy string),
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	for province, flag := range m.mapLocked(played).provinces[asset] {
		visit(province, flag)
	}
	return nil
}

// StoredProvinces is the world's.
func (m *MemoryPersistence) StoredProvinces(asset string) map[clicks.ProvinceID]string {
	return m.StoredProvincesOn(clicks.World, asset)
}

func (m *MemoryPersistence) StoredProvincesOn(played clicks.MapID, asset string) map[clicks.ProvinceID]string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.mapLocked(played).provinces[asset])
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

func (m *MemoryPersistence) LoadOutFlags(_ context.Context, played clicks.MapID, visit func(out clicks.OutFlag)) error {
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
func (m *MemoryPersistence) StoredOutFlagsOn(played clicks.MapID) map[string]clicks.OutFlag {
	m.mu.Lock()
	defer m.mu.Unlock()

	return maps.Clone(m.mapLocked(played).out)
}

func (m *MemoryPersistence) Save(
	_ context.Context, played clicks.Stage, tiles []Tile, provinces []Province, out []clicks.OutFlag,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failing != nil {
		return m.failing
	}
	if len(tiles) > 0 {
		m.saves = append(m.saves, slices.Clone(tiles))
	}
	kept := m.writeLocked(played)
	for _, flag := range out {
		if _, ok := kept.out[flag.Flag]; !ok {
			kept.out[flag.Flag] = flag
		}
	}
	for _, province := range provinces {
		if kept.provinces[province.Asset] == nil {
			kept.provinces[province.Asset] = map[clicks.ProvinceID]string{}
		}
		kept.provinces[province.Asset][province.ID] = province.FortifiedBy
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

func (m *MemoryPersistence) Carry(
	ctx context.Context, onto clicks.Stage, tiles []Tile, mines []clicks.MineSnapshot, keep func(ctx context.Context) (bool, error),
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
	carried := newMemoryMap()
	carried.board = onto.Board
	carried.out = m.mapLocked(onto.Map).out
	for _, tile := range tiles {
		carried.rows[tile.ID] = tile.Owner
		if tile.Shields > 0 {
			carried.shields[tile.ID] = tile.Shields
		}
		if tile.Holder != cpsession.NoAccount {
			carried.holders[tile.ID] = tile.Holder
		}
	}
	for _, mine := range mines {
		carried.mines[mine.Tile] = mine
	}
	m.maps[onto.Map] = carried
	return true, nil
}

func (m *MemoryPersistence) Open(
	ctx context.Context, played clicks.Stage, tiles []Tile, keep func(ctx context.Context) (bool, error),
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
	opened := m.writeLocked(played)
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
		for tile, mine := range kept.mines {
			if mine.Account == account {
				delete(kept.mines, tile)
			}
		}
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
