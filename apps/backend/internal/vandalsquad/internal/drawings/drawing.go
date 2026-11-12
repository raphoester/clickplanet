package drawings

import (
	"math"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

// A tile of the map, as planet numbers it.
type TileID uint32

// On the unit globe, planet's: y is the north pole.
type Point struct{ X, Y, Z float64 }

// One tile of a drawing: where it is, and the tiles of the map it touches, in the map's order.
type Tile struct {
	ID         TileID
	Point      Point
	Neighbours []TileID
}

// Tiles of one player's paint, of one flag, that touch each other, in the order of their ids. A tile's neighbours that
// are not in the drawing are left out of what it reads.
type Drawing struct {
	tiles []Tile
	index map[TileID]int
}

func NewDrawing(tiles []Tile) Drawing {
	sorted := slices.Clone(tiles)
	slices.SortFunc(sorted, func(a, b Tile) int { return int(a.ID) - int(b.ID) })
	sorted = slices.CompactFunc(sorted, func(a, b Tile) bool { return a.ID == b.ID })

	index := make(map[TileID]int, len(sorted))
	for i, tile := range sorted {
		index[tile.ID] = i
	}
	return Drawing{tiles: sorted, index: index}
}

func (d Drawing) Size() int {
	return len(d.tiles)
}

func (d Drawing) Tiles() []Tile {
	return slices.Clone(d.tiles)
}

func (d Drawing) IDs() []TileID {
	ids := make([]TileID, len(d.tiles))
	for i, tile := range d.tiles {
		ids[i] = tile.ID
	}
	return ids
}

// Each tile's neighbours inside the drawing, by their place in it, in the map's order.
func (d Drawing) Links() [][]int {
	links := make([][]int, len(d.tiles))
	for i, tile := range d.tiles {
		for _, neighbour := range tile.Neighbours {
			if j, ok := d.index[neighbour]; ok {
				links[i] = append(links[i], j)
			}
		}
	}
	return links
}

// A point on the screen of a camera that looks at the drawing's middle with the north up: x east, y north.
type Flat struct{ X, Y float64 }

// The drawing as the globe shows it, around its middle.
func (d Drawing) Flats() []Flat {
	var middle Point
	for _, tile := range d.tiles {
		middle = Point{X: middle.X + tile.Point.X, Y: middle.Y + tile.Point.Y, Z: middle.Z + tile.Point.Z}
	}
	centre, ok := unit(middle)
	if !ok {
		return make([]Flat, len(d.tiles))
	}
	up := Point{Y: 1}
	east, ok := unit(cross(up, centre))
	if !ok {
		// A drawing on a pole has no east: any direction across it does.
		east = Point{X: 1}
	}
	north, ok := unit(Point{
		X: up.X - dot(up, centre)*centre.X, Y: up.Y - dot(up, centre)*centre.Y, Z: up.Z - dot(up, centre)*centre.Z,
	})
	if !ok {
		north = cross(centre, east)
	}

	flats := make([]Flat, len(d.tiles))
	for i, tile := range d.tiles {
		flats[i] = Flat{X: dot(tile.Point, east), Y: dot(tile.Point, north)}
	}
	return flats
}

func dot(a, b Point) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func cross(a, b Point) Point {
	return Point{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}
}

func unit(p Point) (Point, bool) {
	length := math.Sqrt(dot(p, p))
	if length == 0 || math.IsNaN(length) {
		return Point{}, false
	}
	return Point{X: p.X / length, Y: p.Y / length, Z: p.Z / length}, true
}

// A tile as the map holds it now: its flag, empty for nobody's, and the player whose take holds it,
// cpsession.NoAccount for none.
type MapTile struct {
	Tile
	Flag   string
	Holder cpsession.AccountID
}
