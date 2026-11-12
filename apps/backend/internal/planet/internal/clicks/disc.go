package clicks

import "slices"

// Disc is every tile within rings steps of the centre, the centre left out, nearest ring first.
func Disc(neighbours Neighbourhood, centre uint32, rings int) []uint32 {
	seen := []uint32{centre}
	ring := []uint32{centre}
	for range rings {
		var next []uint32
		for _, tile := range ring {
			for _, neighbour := range neighbours.Neighbours(tile) {
				if slices.Contains(seen, neighbour) {
					continue
				}
				seen = append(seen, neighbour)
				next = append(next, neighbour)
			}
		}
		ring = next
	}

	return seen[1:]
}
