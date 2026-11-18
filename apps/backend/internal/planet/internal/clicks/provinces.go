package clicks

import "fmt"

// ProvinceID is a row of a board's provinces file: the ground fortify plays on, inside one landmass (docs/provinces.md).
type ProvinceID uint16

const NoProvince ProvinceID = 0

// Provinces are a board's provinces: the province of each tile, and the landmass and the tiles of each. A board with
// none (earth-1) fortifies nothing.
type Provinces struct {
	asset      string
	ofTile     []ProvinceID
	landmasses []LandmassID
	starts     []uint32
	members    []uint32
}

// NoProvinces are the provinces of a board that has none: every tile is in NoProvince.
func NoProvinces() *Provinces {
	return &Provinces{landmasses: []LandmassID{NoLandmass}, starts: []uint32{0, 0}}
}

// NewProvinces refuses provinces cut on other borders: a tile in no province or in one past the table, a province on
// another landmass than its tile's, a province 0 that is not none, or a count that is not its tiles'. ofTile is each
// tile's province, in tile order; landmasses and sizes are each province's, NoProvince first.
func NewProvinces(asset string, borders *Borders, ofTile, landmasses []uint16, sizes []uint32) (*Provinces, error) {
	if uint32(len(ofTile)) != borders.Tiles() { //nolint:gosec // a uint32 tile count, bounded by the blob.
		return nil, fmt.Errorf("%s holds %d tiles, the borders %d", asset, len(ofTile), borders.Tiles())
	}
	if len(landmasses) == 0 || len(sizes) != len(landmasses) {
		return nil, fmt.Errorf("%s names %d landmasses for %d counts", asset, len(landmasses), len(sizes))
	}
	if landmasses[0] != uint16(NoLandmass) || sizes[0] != 0 {
		return nil, fmt.Errorf("%s: province 0 must be none", asset)
	}

	indexed := make([]ProvinceID, len(ofTile)+1)
	of := make([]LandmassID, len(landmasses))
	for k, landmass := range landmasses {
		of[k] = LandmassID(landmass)
	}
	for i, province := range ofTile {
		tile := uint32(i + 1) //nolint:gosec // one entry per tile, bounded by the blob.
		if province == uint16(NoProvince) || int(province) >= len(landmasses) {
			return nil, fmt.Errorf("%s puts tile %d in province %d, of 1 to %d", asset, tile, province, len(landmasses)-1)
		}
		if on := borders.LandmassOf(tile); on != of[province] {
			return nil, fmt.Errorf("%s puts tile %d of landmass %d in province %d, of landmass %d", asset, tile, on,
				province, of[province])
		}
		indexed[tile] = ProvinceID(province)
	}

	provinces := newProvinces(asset, indexed, of)
	for k, size := range sizes {
		if counted := len(provinces.TilesOf(ProvinceID(k))); uint32(counted) != size { //nolint:gosec // provinces are uint16 in the file.
			return nil, fmt.Errorf("%s says province %d holds %d tiles, it holds %d", asset, k, size, counted)
		}
	}

	return provinces, nil
}

// newProvinces indexes the tiles of each province: ofTile[0] is no tile's.
func newProvinces(asset string, ofTile []ProvinceID, landmasses []LandmassID) *Provinces {
	starts, members := membersOf(ofTile, len(landmasses))
	return &Provinces{asset: asset, ofTile: ofTile, landmasses: landmasses, starts: starts, members: members}
}

// Count is how many provinces the board has, none aside.
func (p *Provinces) Count() int {
	return len(p.landmasses) - 1
}

func (p *Provinces) Asset() string {
	return p.asset
}

func (p *Provinces) tiles() uint32 {
	return uint32(max(len(p.ofTile)-1, 0)) //nolint:gosec // one entry per tile, bounded by the file.
}

// ProvinceOf is the tile's province: NoProvince for a tile past the board, or on a board with no provinces.
func (p *Provinces) ProvinceOf(tile uint32) ProvinceID {
	if int(tile) >= len(p.ofTile) {
		return NoProvince
	}
	return p.ofTile[tile]
}

// TilesOf is a window into the provinces' own table: read it, never write it. NoProvince holds the tiles in none.
func (p *Provinces) TilesOf(province ProvinceID) []uint32 {
	if int(province) >= len(p.landmasses) {
		return nil
	}
	return p.members[p.starts[province]:p.starts[province+1]]
}

// membersOf lists the tiles of each of count rows, row by row in tile order, and where each row starts: ofTile[0] is no
// tile's.
func membersOf[Row ~uint16](ofTile []Row, count int) (starts, members []uint32) {
	sizes := make([]uint32, count)
	for _, row := range ofTile[1:] {
		sizes[row]++
	}

	starts = make([]uint32, count+1)
	for row, size := range sizes {
		starts[row+1] = starts[row] + size
	}

	members = make([]uint32, len(ofTile)-1)
	next := make([]uint32, count)
	copy(next, starts)
	for tile := 1; tile < len(ofTile); tile++ {
		row := ofTile[tile]
		members[next[row]] = uint32(tile) //nolint:gosec // one entry per tile, bounded by the file.
		next[row]++
	}
	return starts, members
}
