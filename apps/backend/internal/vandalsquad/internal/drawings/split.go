package drawings

import "slices"

// The drawings in some tiles of one paint: the groups of them that touch each other, smallest id first.
func Split(tiles []Tile) []Drawing {
	whole := NewDrawing(tiles)
	links := whole.Links()

	var split []Drawing
	seen := make([]bool, whole.Size())
	for start := range whole.Size() {
		if seen[start] {
			continue
		}
		group := []int{start}
		seen[start] = true
		for next := 0; next < len(group); next++ {
			for _, neighbour := range links[group[next]] {
				if !seen[neighbour] {
					seen[neighbour] = true
					group = append(group, neighbour)
				}
			}
		}
		part := make([]Tile, len(group))
		for i, tile := range group {
			part[i] = whole.tiles[tile]
		}
		split = append(split, NewDrawing(part))
	}
	return split
}

// The tiles of the map around the drawing: each one it touches that is not in it, once, smallest first.
func (d Drawing) Rim() []TileID {
	var rim []TileID
	for _, tile := range d.tiles {
		for _, neighbour := range tile.Neighbours {
			if _, in := d.index[neighbour]; !in {
				rim = append(rim, neighbour)
			}
		}
	}
	slices.Sort(rim)
	return slices.Compact(rim)
}

// A drawing shows when most of the land around it wears another flag. Tiles a player paints back to the flag all
// around them are a clean-up, not a drawing: players painting over a swastika on their own ground leave fresh tiles of
// its very shape, and the one around a real drawing is all another flag's.
const maxSameAround = 0.5

func Visible(flag string, around []string) bool {
	if len(around) == 0 {
		return true
	}
	same := 0
	for _, other := range around {
		if other == flag {
			same++
		}
	}
	return float64(same) <= maxSameAround*float64(len(around))
}
