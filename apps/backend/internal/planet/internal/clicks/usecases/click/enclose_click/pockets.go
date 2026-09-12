package enclose_click

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// pocket is a closed shape: the tiles inside it that are not the caller's, and
// the caller's tiles around them.
type pocket struct {
	filled []uint32
	wall   []uint32
}

// findPockets looks for every shape the clicked tile closes.
//
// It never looks for the outline. On a sphere every loop cuts the planet in two,
// and both halves are "inside" it — so it looks for a small inside instead. From
// each neighbour of the clicked tile that is not the country's, it floods over
// tiles that are not the country's. A flood that runs out of tiles before
// passing maxTiles found a pocket; one that passes it is open ground, or a shape
// too big to take, and takes nothing.
//
// A pocket must be walled by the country's tiles alone. A tile on the edge of
// the map — a coast, a lake shore — has fewer than six neighbours, so a flood
// that reaches one is open: cutting off the tip of a peninsula, or owning all
// but one tile of a small island, is not a closed shape. The twelve corners of
// the lattice also have five neighbours and read as an edge too; that is twelve
// tiles out of a quarter of a million.
//
// Nothing here holds a lock across the search, so a tile can change while it
// runs. The worst that does is fill a pocket that opened a moment ago, or miss
// one that closed.
func findPockets(closing uint32, country string, maxTiles int, owners Owners, neighbours Neighbours) []pocket {
	var pockets []pocket

	// Every tile a flood reached, found or not. A flood that passed the limit
	// only saw part of its region, but all of that part is one region, so any
	// other start inside it would pass the limit too.
	seen := map[uint32]bool{}

	for _, start := range neighbours.Neighbours(closing) {
		if seen[start] || owns(owners, start, country) {
			continue
		}

		if found, ok := flood(start, country, maxTiles, owners, neighbours, seen); ok {
			pockets = append(pockets, found)
		}
	}

	return pockets
}

func flood(start uint32, country string, maxTiles int, owners Owners, neighbours Neighbours, seen map[uint32]bool) (pocket, bool) {
	found := pocket{filled: []uint32{start}}
	seen[start] = true

	inside := map[uint32]bool{start: true}
	walls := map[uint32]bool{}

	// The filled slice is the queue: breadth first, so the tiles come out
	// nearest the start — which touches the clicked tile — first.
	for next := 0; next < len(found.filled); next++ {
		around := neighbours.Neighbours(found.filled[next])
		if len(around) < clicks.MaxDegree {
			return pocket{}, false
		}

		for _, tile := range around {
			switch {
			case inside[tile] || walls[tile]:
				continue
			case owns(owners, tile, country):
				walls[tile] = true
				found.wall = append(found.wall, tile)
			default:
				if len(found.filled) == maxTiles {
					return pocket{}, false
				}

				inside[tile] = true
				seen[tile] = true
				found.filled = append(found.filled, tile)
			}
		}
	}

	return found, true
}

func owns(owners Owners, tile uint32, country string) bool {
	owner, _ := owners.Owner(tile)
	return owner == country
}
