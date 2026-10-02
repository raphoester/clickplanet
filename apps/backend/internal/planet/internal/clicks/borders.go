package clicks

import "fmt"

type Borders struct {
	regions []uint16
	codes   []string
}

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

func (b *Borders) CountryOf(tile uint32) string {
	if int(tile) >= len(b.regions) {
		return ""
	}

	return b.codes[b.regions[tile]]
}

func (b *Borders) Tiles() uint32 {
	return uint32(len(b.regions) - 1) //nolint:gosec // one entry per tile, bounded by the blob.
}
