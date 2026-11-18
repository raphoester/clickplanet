package clicks

import "fmt"

// ProvinceID is a row of a board's provinces file: the ground fortify plays on, inside one landmass (docs/provinces.md).
type ProvinceID uint16

const NoProvince ProvinceID = 0

// Provinces are a board's provinces: the province of each tile, and the landmass and tile count of each. No rule reads
// them yet. earth-1 has none.
type Provinces struct {
	asset      string
	ofTile     []ProvinceID
	landmasses []LandmassID
	sizes      []uint32
}

// NoProvinces are the provinces of a board that has none: every tile is in NoProvince.
func NoProvinces() *Provinces {
	return &Provinces{landmasses: []LandmassID{NoLandmass}, sizes: []uint32{0}}
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

	counted := make([]uint32, len(landmasses))
	provinces := &Provinces{
		asset:      asset,
		ofTile:     make([]ProvinceID, len(ofTile)+1),
		landmasses: make([]LandmassID, len(landmasses)),
		sizes:      sizes,
	}
	for k, landmass := range landmasses {
		provinces.landmasses[k] = LandmassID(landmass)
	}
	for i, province := range ofTile {
		tile := uint32(i + 1) //nolint:gosec // one entry per tile, bounded by the blob.
		if province == uint16(NoProvince) || int(province) >= len(landmasses) {
			return nil, fmt.Errorf("%s puts tile %d in province %d, of 1 to %d", asset, tile, province, len(landmasses)-1)
		}
		if on := borders.LandmassOf(tile); on != provinces.landmasses[province] {
			return nil, fmt.Errorf("%s puts tile %d of landmass %d in province %d, of landmass %d", asset, tile, on,
				province, provinces.landmasses[province])
		}
		provinces.ofTile[tile] = ProvinceID(province)
		counted[province]++
	}
	for k, size := range sizes {
		if counted[k] != size {
			return nil, fmt.Errorf("%s says province %d holds %d tiles, it holds %d", asset, k, size, counted[k])
		}
	}

	return provinces, nil
}

// Count is how many provinces the board has, none aside.
func (p *Provinces) Count() int {
	return len(p.landmasses) - 1
}

func (p *Provinces) Asset() string {
	return p.asset
}
