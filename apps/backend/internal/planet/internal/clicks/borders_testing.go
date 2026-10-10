//go:build testing

package clicks

import "fmt"

// BordersOf puts landmasses[k] on landmass k+1, ground of "l<k+1>", and every other tile on no landmass.
func BordersOf(tiles uint32, landmasses ...[]uint32) *Borders {
	regions := make([]uint16, tiles)
	codes := make([]string, 1, 1+len(landmasses))
	for k, members := range landmasses {
		codes = append(codes, fmt.Sprintf("l%d", k+1))
		for _, tile := range members {
			regions[tile-1] = uint16(k + 1) //nolint:gosec // a handful of landmasses in a test.
		}
	}

	borders, err := NewBorders("test-borders", regions, codes)
	if err != nil {
		panic(err)
	}

	return borders
}
