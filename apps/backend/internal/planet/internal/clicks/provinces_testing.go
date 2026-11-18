//go:build testing

package clicks

// ProvinceOf is the tile's province: NoProvince for a tile past the board, or on a board with no provinces.
func (p *Provinces) ProvinceOf(tile uint32) ProvinceID {
	if int(tile) >= len(p.ofTile) {
		return NoProvince
	}
	return p.ofTile[tile]
}

// LandmassOf is the landmass the province is inside.
func (p *Provinces) LandmassOf(province ProvinceID) LandmassID {
	if int(province) >= len(p.landmasses) {
		return NoLandmass
	}
	return p.landmasses[province]
}

// Size is how many tiles the province holds.
func (p *Provinces) Size(province ProvinceID) uint32 {
	if int(province) >= len(p.sizes) {
		return 0
	}
	return p.sizes[province]
}
