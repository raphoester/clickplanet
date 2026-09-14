package clicks

import "fmt"

// Borders says which country's ground a tile sits on. It is geography, not ownership: it never changes.
type Borders struct {
	// Indexed by tile id; slot 0 is unused, like the tile storage's.
	regions []uint16
	codes   []string
}

// NewBorders takes one region per tile, 0-indexed by blob position, and the code of each region; region 0 is no country.
func NewBorders(regions []uint16, codes []string) (*Borders, error) {
	if len(codes) == 0 || codes[0] != "" {
		return nil, fmt.Errorf("region 0 must be no country")
	}

	indexed := make([]uint16, len(regions)+1)
	for i, region := range regions {
		if int(region) >= len(codes) {
			return nil, fmt.Errorf("tile %d is in region %d, past the %d regions named", i+1, region, len(codes))
		}
		indexed[i+1] = region
	}

	return &Borders{regions: indexed, codes: codes}, nil
}

// CountryOf is empty for a tile outside every country and for a tile past the end of the map.
func (b *Borders) CountryOf(tile uint32) string {
	if int(tile) >= len(b.regions) {
		return ""
	}

	return b.codes[b.regions[tile]]
}

func (b *Borders) Tiles() uint32 {
	return uint32(len(b.regions) - 1) //nolint:gosec // one entry per tile, bounded by the blob.
}
