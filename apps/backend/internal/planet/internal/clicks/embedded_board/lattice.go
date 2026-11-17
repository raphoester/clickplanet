package embedded_board

import (
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// A lattice point's canonical id: the corners, then the 30 edges, then each face's inside, row by row.
type canonicalID int

const (
	cornerCount = 12
	edgeCount   = 30
	faceCount   = 20
)

var t = (1 + math.Sqrt(5)) / 2

// Not normalised: THREE lerps the flat face first, so normalising here moves every tile.
var icosahedronCorners = [cornerCount]clicks.Vec3{
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

var icosahedronFaces = [faceCount][3]int{
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

// The ±1 exchanges of a point's barycentric coordinates, as (i, j) steps on its face.
var latticeMoves = [6][2]int{
	{0, +1}, {0, -1}, {+1, 0}, {-1, 0}, {+1, -1}, {-1, +1},
}

func latticeSize(cols int) int {
	return 10*cols*cols + 2
}

// The id of the point `step` along an edge is at + step*by.
type edgeIDs struct {
	at canonicalID
	by int
}

type faceIDs struct {
	corners [3]canonicalID
	// a to b, a to c, b to c.
	edges  [3]edgeIDs
	inside canonicalID
}

func latticeFaceIDs(cols int) [faceCount]faceIDs {
	var edgeOf [cornerCount][cornerCount]int
	for p := range edgeOf {
		for q := range edgeOf[p] {
			edgeOf[p][q] = -1
		}
	}
	edges := 0
	for _, corners := range icosahedronFaces {
		a, b, c := corners[0], corners[1], corners[2]
		for _, pair := range [3][2]int{{a, b}, {b, c}, {a, c}} {
			if edgeOf[pair[0]][pair[1]] != -1 {
				continue
			}
			edgeOf[pair[0]][pair[1]], edgeOf[pair[1]][pair[0]] = edges, edges
			edges++
		}
	}

	edge := func(from, to int) edgeIDs {
		first := canonicalID(cornerCount + edgeOf[from][to]*(cols-1) - 1)
		if from < to {
			return edgeIDs{at: first, by: 1}
		}
		return edgeIDs{at: first + canonicalID(cols), by: -1}
	}
	insideFirst := canonicalID(cornerCount + edgeCount*(cols-1))
	insidePerFace := canonicalID((cols - 2) * (cols - 1) / 2)

	var ids [faceCount]faceIDs
	for face, corners := range icosahedronFaces {
		a, b, c := corners[0], corners[1], corners[2]
		ids[face] = faceIDs{
			corners: [3]canonicalID{canonicalID(a), canonicalID(b), canonicalID(c)},
			edges:   [3]edgeIDs{edge(a, b), edge(a, c), edge(b, c)},
			inside:  insideFirst + canonicalID(face)*insidePerFace,
		}
	}
	return ids
}

func (f faceIDs) of(cols, i, j int) canonicalID {
	switch {
	case i == 0 && j == 0:
		return f.corners[0]
	case i == 0 && j == cols:
		return f.corners[1]
	case i == cols:
		return f.corners[2]
	case i == 0:
		return f.edges[0].at + canonicalID(j*f.edges[0].by)
	case j == 0:
		return f.edges[1].at + canonicalID(i*f.edges[1].by)
	case j == cols-i:
		return f.edges[2].at + canonicalID(i*f.edges[2].by)
	default:
		return f.inside + canonicalID((i-1)*(cols-2)-(i-1)*(i-2)/2+j-1)
	}
}

// walkLattice visits each point once, in three's order: the first time three's triangles reach it.
func walkLattice(cols int, visit func(face, i, j int, id canonicalID)) {
	ids := latticeFaceIDs(cols)
	seen := make([]bool, latticeSize(cols))
	push := func(face, i, j int) {
		id := ids[face].of(cols, i, j)
		if seen[id] {
			return
		}
		seen[id] = true
		visit(face, i, j, id)
	}

	for face := range faceCount {
		for i := range cols {
			for j := range 2*(cols-i) - 1 {
				k := j / 2
				if j%2 == 0 {
					push(face, i, k+1)
					push(face, i+1, k)
					push(face, i, k)
				} else {
					push(face, i, k+1)
					push(face, i+1, k+1)
					push(face, i+1, k)
				}
			}
		}
	}
}

// latticePosition is three's, to the bit, as every app rebuilds it.
func latticePosition(cols, face, i, j int) [3]float32 {
	a := icosahedronCorners[icosahedronFaces[face][0]]
	b := icosahedronCorners[icosahedronFaces[face][1]]
	c := icosahedronCorners[icosahedronFaces[face][2]]

	down := float64(i) / float64(cols)
	x, y, z := lerp(a.X, c.X, down), lerp(a.Y, c.Y, down), lerp(a.Z, c.Z, down)
	if j != 0 || i != cols {
		across := float64(j) / float64(cols-i)
		x = lerp(x, lerp(b.X, c.X, down), across)
		y = lerp(y, lerp(b.Y, c.Y, down), across)
		z = lerp(z, lerp(b.Z, c.Z, down), across)
	}

	scale := 1 / math.Sqrt(float64(x*x)+float64(y*y)+float64(z*z))
	return [3]float32{float32(x * scale), float32(y * scale), float32(z * scale)}
}

// The conversion rounds the product on its own: fused into the add, it would round once and move a tile by an ulp.
func lerp(from, to, at float64) float64 {
	return from + float64((to-from)*at)
}

func positionsHash(positions []float32) uint32 {
	hash := uint32(0x811c9dc5)
	for _, position := range positions {
		hash = (hash ^ math.Float32bits(position)) * 0x01000193
	}
	return hash
}
