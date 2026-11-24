package clicks

import (
	"math"
	"slices"
	"strings"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

// Pin is a point in degrees, never a tile: each board numbers its tiles its own way.
type Pin struct {
	Latitude  float64
	Longitude float64
}

// As the tile positions: y is the north pole, and longitude 0 is on -x.
func (p Pin) Point() Vec3 {
	latitude, longitude := p.Latitude*math.Pi/180, p.Longitude*math.Pi/180
	return Vec3{
		X: -math.Cos(latitude) * math.Cos(longitude),
		Y: math.Sin(latitude),
		Z: math.Cos(latitude) * math.Sin(longitude),
	}
}

type Pinned struct {
	Flag string
	Pin  Pin
}

// In spacings: each search after the first looks twice as far.
const seatSearch = 2

// Seats sits each flag on the land tile nearest its pin that is no capital and touches none, in the order of the codes.
func (g *Ground) Seats(pinned []Pinned) []Capital {
	taken := cpcolls.NewSet[uint32]()
	for _, capital := range g.capitals.All() {
		taken.Add(capital.Tile)
	}

	ordered := slices.SortedFunc(slices.Values(pinned), func(a, b Pinned) int { return strings.Compare(a.Flag, b.Flag) })
	seats := make([]Capital, 0, len(ordered))
	for i, flag := range ordered {
		if g.capitals.Has(flag.Flag) || (i > 0 && ordered[i-1].Flag == flag.Flag) {
			continue
		}
		tile, ok := g.seatOf(flag.Pin.Point(), taken)
		if !ok {
			continue
		}
		taken.Add(tile)
		seats = append(seats, Capital{Country: flag.Flag, Tile: tile})
	}

	return seats
}

func (g *Ground) seatOf(point Vec3, taken *cpcolls.Set[uint32]) (uint32, bool) {
	radius := max(seatSearch*g.spacing, 1e-3)
	for {
		best, bestAngle := uint32(0), math.Inf(1)
		for _, tile := range g.geography.Within(point, radius) {
			if !g.free(tile, taken) {
				continue
			}
			position, _ := g.geography.Position(tile)
			if a := angle(point.Unit(), position); a < bestAngle {
				best, bestAngle = tile, a
			}
		}
		if best != 0 {
			return best, true
		}
		if radius >= math.Pi {
			return 0, false
		}
		radius = min(2*radius, math.Pi)
	}
}

func (g *Ground) free(tile uint32, taken *cpcolls.Set[uint32]) bool {
	if taken.Contains(tile) || g.borders.CountryOf(tile) == "" {
		return false
	}
	return !slices.ContainsFunc(g.geography.Neighbours(tile), taken.Contains)
}
