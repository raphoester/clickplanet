//go:build testing

package clicks

import "fmt"

// LandmassOf is the landmass the province is inside.
func (p *Provinces) LandmassOf(province ProvinceID) LandmassID {
	if int(province) >= len(p.landmasses) {
		return NoLandmass
	}
	return p.landmasses[province]
}

// Size is how many tiles the province holds.
func (p *Provinces) Size(province ProvinceID) uint32 {
	return uint32(len(p.TilesOf(province))) //nolint:gosec // provinces are bounded by the file.
}

// ProvincesOf puts provinces[k] in province k+1, of the landmass of its first tile, and every other tile in none.
func ProvincesOf(asset string, borders *Borders, provinces ...[]uint32) *Provinces {
	ofTile := make([]ProvinceID, borders.Tiles()+1)
	landmasses := make([]LandmassID, 1, 1+len(provinces))
	for k, members := range provinces {
		province := ProvinceID(k + 1) //nolint:gosec // a handful of provinces in a test.
		landmass := borders.LandmassOf(members[0])
		for _, tile := range members {
			if borders.LandmassOf(tile) != landmass {
				panic(fmt.Sprintf("province %d crosses landmasses %d and %d", province, landmass, borders.LandmassOf(tile)))
			}
			ofTile[tile] = province
		}
		landmasses = append(landmasses, landmass)
	}
	return newProvinces(asset, ofTile, landmasses)
}

// ProvincesOfLandmasses makes each landmass one province of the same id: what a test of a ground with no provinces of
// its own needs.
func ProvincesOfLandmasses(borders *Borders) *Provinces {
	ofTile := make([]ProvinceID, borders.Tiles()+1)
	for tile := uint32(1); tile <= borders.Tiles(); tile++ {
		ofTile[tile] = ProvinceID(borders.LandmassOf(tile))
	}
	landmasses := make([]LandmassID, borders.Landmasses())
	for landmass := range landmasses {
		landmasses[landmass] = LandmassID(landmass) //nolint:gosec // landmasses are uint16 in the blob.
	}
	return newProvinces("test-provinces", ofTile, landmasses)
}
