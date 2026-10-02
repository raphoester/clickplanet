package embedded_geodesic_map

import (
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const Detail = 300

const Cols = Detail + 1

var t = (1 + math.Sqrt(5)) / 2

// Not normalised: THREE lerps the flat face first, so normalising here moves every tile.
var icosahedronVertices = [12]clicks.Vec3{
	{X: -1, Y: t},
	{X: 1, Y: t},
	{X: -1, Y: -t},
	{X: 1, Y: -t},
	{Y: -1, Z: t},
	{Y: 1, Z: t},
	{Y: -1, Z: -t},
	{Y: 1, Z: -t},
	{X: t, Z: -1},
	{X: t, Z: 1},
	{X: -t, Z: -1},
	{X: -t, Z: 1},
}

var icosahedronFaces = [20][3]int{
	{0, 11, 5},
	{0, 5, 1},
	{0, 1, 7},
	{0, 7, 10},
	{0, 10, 11},
	{1, 5, 9},
	{5, 11, 4},
	{11, 10, 2},
	{10, 7, 6},
	{7, 1, 8},
	{3, 9, 4},
	{3, 4, 2},
	{3, 2, 6},
	{3, 6, 8},
	{3, 8, 9},
	{4, 9, 5},
	{2, 4, 11},
	{6, 2, 10},
	{8, 6, 7},
	{9, 8, 1},
}

func latticePosition(a, b, c clicks.Vec3, p, q, r int) clicks.Vec3 {
	combined := clicks.Vec3{
		X: a.X*float64(p) + b.X*float64(q) + c.X*float64(r),
		Y: a.Y*float64(p) + b.Y*float64(q) + c.Y*float64(r),
		Z: a.Z*float64(p) + b.Z*float64(q) + c.Z*float64(r),
	}

	length := math.Sqrt(combined.X*combined.X + combined.Y*combined.Y + combined.Z*combined.Z)
	if length == 0 {
		return combined
	}

	return clicks.Vec3{X: combined.X / length, Y: combined.Y / length, Z: combined.Z / length}
}

var latticeMoves = [6][2]int{
	{0, +1}, {0, -1}, {+1, 0}, {-1, 0}, {+1, -1}, {-1, +1},
}

const faceVertices = (Cols + 1) * (Cols + 2) / 2

func faceIndex(i, j int) int {
	return i*(Cols+1) - (i*(i-1))/2 + j
}
