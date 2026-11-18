package embedded_board

import (
	"fmt"
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

// walkCanonical visits each point once, in canonical id order: the 12 corners, then the 30 edges step by step from their
// lower corner, then each face's inside, row by row. A point on an edge or a corner comes out the same from every face it
// is on, to the bit, so the face it is reached from does not matter.
func walkCanonical(cols int, visit func(face, i, j int, id canonicalID)) {
	ids := latticeFaceIDs(cols)
	// (i, j) of a face's corners a, b, c.
	at := [3][2]int{{0, 0}, {0, cols}, {cols, 0}}
	place := func(face, corner int) (int, int, bool) {
		for k, of := range icosahedronFaces[face] {
			if of == corner {
				return at[k][0], at[k][1], true
			}
		}
		return 0, 0, false
	}

	id := canonicalID(0)
	for corner := range cornerCount {
		for face := range faceCount {
			if i, j, ok := place(face, corner); ok {
				visit(face, i, j, id)
				id++
				break
			}
		}
	}

	for _, edge := range latticeEdgeList() {
		for face := range faceCount {
			fi, fj, fromOK := place(face, edge[0])
			ti, tj, toOK := place(face, edge[1])
			if !fromOK || !toOK {
				continue
			}
			di, dj := (ti-fi)/cols, (tj-fj)/cols
			for step := 1; step < cols; step++ {
				visit(face, fi+di*step, fj+dj*step, id)
				id++
			}
			break
		}
	}

	for face := range faceCount {
		if id != ids[face].inside {
			panic(fmt.Sprintf("face %d's inside starts at %d, not %d", face, ids[face].inside, id))
		}
		for i := 1; i < cols-1; i++ {
			for j := 1; j < cols-i; j++ {
				visit(face, i, j, id)
				id++
			}
		}
	}
}

// latticeEdgeList is the 30 edges, as their two corners, the lower first, in the order their canonical ids run.
func latticeEdgeList() [][2]int {
	var seen [cornerCount][cornerCount]bool
	edges := make([][2]int, 0, edgeCount)
	for _, corners := range icosahedronFaces {
		a, b, c := corners[0], corners[1], corners[2]
		for _, pair := range [3][2]int{{a, b}, {b, c}, {a, c}} {
			low, high := min(pair[0], pair[1]), max(pair[0], pair[1])
			if seen[low][high] {
				continue
			}
			seen[low][high] = true
			edges = append(edges, [2]int{low, high})
		}
	}
	return edges
}

// latticeWarp is a board's warp of a point's barycentric weights (docs/boards.md, "The geometry").
type latticeWarp struct {
	a, b, c float64
}

// latticeShape is the lattice a board's manifest describes: three's (no warp, no rotation, three's order) or a warped
// and rotated one in canonical order.
type latticeShape struct {
	cols    int
	warp    *latticeWarp
	corners [cornerCount]clicks.Vec3
	threes  bool
	order   string
}

const (
	threesOrder    = "three"
	canonicalOrder = "canonical"
)

// walk visits each point once in the board's tile order: three's walk, or canonical.
func (s latticeShape) walk(visit func(face, i, j int, id canonicalID)) {
	if s.order == canonicalOrder {
		walkCanonical(s.cols, visit)
		return
	}
	walkLattice(s.cols, visit)
}

// position is (face, i, j)'s place on the sphere, as the page rebuilds it, to the bit.
func (s latticeShape) position(face, i, j int) [3]float32 {
	if s.threes {
		return latticePosition(s.cols, face, i, j)
	}
	cols := float64(s.cols)
	wa, wb, wc := float64(s.cols-i-j)/cols, float64(j)/cols, float64(i)/cols
	if s.warp != nil {
		wa, wb, wc = s.warp.weights(wa, wb, wc)
	}
	a := s.corners[icosahedronFaces[face][0]]
	b := s.corners[icosahedronFaces[face][1]]
	c := s.corners[icosahedronFaces[face][2]]

	// (wa·A + wb·B) + wc·C, each product rounded on its own: fused into the add, it would round once and move a tile.
	x := float64(wa*a.X) + float64(wb*b.X) + float64(wc*c.X)
	y := float64(wa*a.Y) + float64(wb*b.Y) + float64(wc*c.Y)
	z := float64(wa*a.Z) + float64(wb*b.Z) + float64(wc*c.Z)
	scale := 1 / math.Sqrt(float64(x*x)+float64(y*y)+float64(z*z))
	return [3]float32{float32(x * scale), float32(y * scale), float32(z * scale)}
}

// weights warps barycentric weights as the page does: h_k = w_k·((1 + a·w_k) + b·(w_k·w_k)) + ((c·wa)·wb)·wc, then each
// over (ha + hb) + hc. Every product is converted on its own, so none is fused into an add.
func (w latticeWarp) weights(wa, wb, wc float64) (float64, float64, float64) {
	shared := float64(float64(float64(w.c*wa)*wb) * wc)
	ha := w.lifted(wa) + shared
	hb := w.lifted(wb) + shared
	hc := w.lifted(wc) + shared
	sum := ha + hb + hc
	return ha / sum, hb / sum, hc / sum
}

// lifted is w·((1 + a·w) + b·(w·w)).
func (w latticeWarp) lifted(of float64) float64 {
	return float64(of * (float64(1+float64(w.a*of)) + float64(w.b*float64(of*of))))
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
