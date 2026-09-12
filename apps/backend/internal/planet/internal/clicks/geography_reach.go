package clicks

import (
	"math"
)

// Spacing is the mean arc between two touching tiles, in radians.
func (g *Geography) Spacing() float64 {
	total, edges := 0.0, 0

	for id := uint32(1); id <= g.stats.Tiles; id++ {
		from, _ := g.Position(id)
		for _, other := range g.Neighbours(id) {
			to, _ := g.Position(other)
			total += angle(from, to)
			edges++
		}
	}

	if edges == 0 {
		return 0
	}

	return total / float64(edges)
}

// Within returns every tile within radius radians of arc of centre, ascending: a true circle, and
// across water as well as land. A straight scan, once per bomb.
func (g *Geography) Within(centre Vec3, radius float64) []uint32 {
	unit, ok := normalize(centre)
	if !ok {
		return nil
	}

	threshold := math.Cos(radius)

	var found []uint32
	for id := uint32(1); id <= g.stats.Tiles; id++ {
		i := int(id-1) * 3
		along := float64(g.positions[i])*unit.X + float64(g.positions[i+1])*unit.Y + float64(g.positions[i+2])*unit.Z
		if along >= threshold {
			found = append(found, id)
		}
	}

	return found
}

// Nearest returns the tile closest to point and the arc to it; a straight scan, once per bomb.
func (g *Geography) Nearest(point Vec3) (uint32, float64) {
	unit, ok := normalize(point)
	if !ok {
		return 0, math.Pi
	}

	best, bestAlong := uint32(0), -2.0
	for id := uint32(1); id <= g.stats.Tiles; id++ {
		i := int(id-1) * 3
		along := float64(g.positions[i])*unit.X + float64(g.positions[i+1])*unit.Y + float64(g.positions[i+2])*unit.Z
		if along > bestAlong {
			best, bestAlong = id, along
		}
	}

	return best, math.Acos(min(bestAlong, 1))
}

func angle(a, b Vec3) float64 {
	return math.Acos(min(max(a.X*b.X+a.Y*b.Y+a.Z*b.Z, -1), 1))
}

func normalize(v Vec3) (Vec3, bool) {
	length := math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
	if length == 0 || math.IsNaN(length) || math.IsInf(length, 0) {
		return Vec3{}, false
	}

	return Vec3{X: v.X / length, Y: v.Y / length, Z: v.Z / length}, true
}

// Position is where tile id sits on the unit sphere.
func (g *Geography) Position(id uint32) (Vec3, bool) {
	if id == 0 || id > g.stats.Tiles {
		return Vec3{}, false
	}

	i := int(id-1) * 3
	return Vec3{
		X: float64(g.positions[i]),
		Y: float64(g.positions[i+1]),
		Z: float64(g.positions[i+2]),
	}, true
}
