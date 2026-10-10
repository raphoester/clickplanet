package clicks

import "fmt"

type LandmassID uint16

const NoLandmass LandmassID = 0

type Borders struct {
	asset   string
	regions []uint16
	codes   []string

	starts  []uint32
	members []uint32
}

func NewBorders(asset string, regions []uint16, codes []string) (*Borders, error) {
	if len(codes) == 0 || codes[0] != "" {
		return nil, fmt.Errorf("region 0 must be no country")
	}

	indexed := make([]uint16, len(regions)+1)
	sizes := make([]uint32, len(codes))
	for i, region := range regions {
		if int(region) >= len(codes) {
			return nil, fmt.Errorf("tile %d is in region %d, past the %d regions named", i+1, region, len(codes))
		}
		indexed[i+1] = region
		sizes[region]++
	}

	starts := make([]uint32, len(codes)+1)
	for region, size := range sizes {
		starts[region+1] = starts[region] + size
	}

	members := make([]uint32, len(regions))
	next := make([]uint32, len(codes))
	copy(next, starts)
	for tile := 1; tile < len(indexed); tile++ {
		region := indexed[tile]
		members[next[region]] = uint32(tile) //nolint:gosec // one entry per tile, bounded by the blob.
		next[region]++
	}

	return &Borders{asset: asset, regions: indexed, codes: codes, starts: starts, members: members}, nil
}

func (b *Borders) CountryOf(tile uint32) string {
	return b.codes[b.LandmassOf(tile)]
}

func (b *Borders) LandmassOf(tile uint32) LandmassID {
	if int(tile) >= len(b.regions) {
		return NoLandmass
	}

	return LandmassID(b.regions[tile])
}

// TilesOf is a window into the borders' own table: read it, never write it.
func (b *Borders) TilesOf(landmass LandmassID) []uint32 {
	if int(landmass) >= len(b.codes) {
		return nil
	}

	return b.members[b.starts[landmass]:b.starts[landmass+1]]
}

func (b *Borders) Landmasses() int {
	return len(b.codes)
}

func (b *Borders) Tiles() uint32 {
	return uint32(len(b.regions) - 1) //nolint:gosec // one entry per tile, bounded by the blob.
}

func (b *Borders) Asset() string {
	return b.asset
}
