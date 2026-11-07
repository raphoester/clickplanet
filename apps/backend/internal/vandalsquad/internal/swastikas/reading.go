// A swastika as players draw it on the map: a centre, four branches, each turning once, all four the same way.
//
// Every rule below was read off real swastikas painted on the map (testdata/drawings.json holds them, and the drawings
// that are none): strokes one tile wide, along the grid's lines or zigzagging across them, arms and hooks of uneven
// length, each turn 60, 90 or 120 degrees on screen. So nothing here measures a right angle or a symmetry; it follows
// the tiles.
package swastikas

import (
	"cmp"
	"math"
	"slices"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
)

const (
	// The smallest real one is 16 tiles; past 150, a drawing is a battle's front, not a symbol.
	minTiles = 12
	maxTiles = 150
	// No tile of a real one has five or six neighbours in it; a blob of conquest has them all through its middle.
	maxThick = 0.05
	// Real turns run from 58 to 121 degrees.
	minTurn = 40.0
	maxTurn = 140.0
	// The four arms leave the centre roughly a quarter turn apart: 46 to 122 degrees on the real ones.
	minGap = 45.0
	maxGap = 135.0
	// The longest branch of a real one is at most twice its shortest; four and six times on a path with stubs and on a
	// wheel with spokes.
	maxUneven = 3.0
	// At most one arm of a real one is a single tile; three of a hook-shaped blob the operator judged none.
	maxShortArms = 2
	// 88 to 97% of a real one's tiles lie on its four branches, 51 to 65% of the scribbles the rest lets through;
	// every tile is on them or next to them.
	minOn    = 0.8
	minCover = 0.85
)

// How a drawing reads as a swastika: its centre, and the share of its tiles on the four branches.
type Reading struct {
	Centre drawings.TileID
	On     float64
}

// The best reading of the drawing as a swastika, if it is one.
func ReadingOf(drawing drawings.Drawing) (Reading, bool) {
	n := drawing.Size()
	if n < minTiles || n > maxTiles {
		return Reading{}, false
	}
	links := drawing.Links()
	thick := 0
	for _, link := range links {
		if len(link) >= 5 {
			thick++
		}
	}
	if float64(thick) > maxThick*float64(n) {
		return Reading{}, false
	}

	flats := drawing.Flats()
	ids := drawing.IDs()
	var best Reading
	found := false
	for centre := range n {
		if len(links[centre]) < 3 {
			continue
		}
		walk := walkFrom(links, centre)
		if len(walk.order) < n {
			return Reading{}, false // not one drawing
		}
		on, ok := readFrom(links, flats, walk, centre)
		if ok && (!found || on > best.On) {
			best, found = Reading{Centre: ids[centre], On: on}, true
		}
	}
	return best, found
}

// The breadth-first walk of the drawing from one tile: each tile's steps from it and the tile it was reached from.
type walk struct {
	steps  []int
	parent []int
	order  []int
}

func walkFrom(links [][]int, start int) walk {
	w := walk{steps: make([]int, len(links)), parent: make([]int, len(links)), order: []int{start}}
	for i := range w.steps {
		w.steps[i] = -1
	}
	w.steps[start], w.parent[start] = 0, -1
	for next := 0; next < len(w.order); next++ {
		tile := w.order[next]
		for _, neighbour := range links[tile] {
			if w.steps[neighbour] < 0 {
				w.steps[neighbour] = w.steps[tile] + 1
				w.parent[neighbour] = tile
				w.order = append(w.order, neighbour)
			}
		}
	}
	return w
}

// From the centre back out to a tile, centre first.
func (w walk) pathTo(tile int) []int {
	var path []int
	for at := tile; at >= 0; at = w.parent[at] {
		path = append(path, at)
	}
	slices.Reverse(path)
	return path
}

// One branch: the arm from the centre to its corner, the hook from the corner to the tip, and the corner's step.
type branch struct {
	arm, hook drawings.Flat
	corner    int
	steps     int
}

// The corner is the tile farthest from the straight line between the centre and the tip.
func branchOf(path []int, flats []drawings.Flat) (branch, bool) {
	start, tip := flats[path[0]], flats[path[len(path)-1]]
	chord := drawings.Flat{X: tip.X - start.X, Y: tip.Y - start.Y}
	length := math.Hypot(chord.X, chord.Y)
	if length < 1e-12 {
		return branch{}, false
	}
	normal := drawings.Flat{X: -chord.Y / length, Y: chord.X / length}
	corner, farthest := 0, -1.0
	for k, tile := range path {
		off := math.Abs((flats[tile].X-start.X)*normal.X + (flats[tile].Y-start.Y)*normal.Y)
		if off > farthest {
			corner, farthest = k, off
		}
	}
	at := flats[path[corner]]
	return branch{
		arm:    drawings.Flat{X: at.X - start.X, Y: at.Y - start.Y},
		hook:   drawings.Flat{X: tip.X - at.X, Y: tip.Y - at.Y},
		corner: corner,
		steps:  len(path) - 1,
	}, true
}

func (b branch) turn() float64 {
	return wrap(angle(b.hook) - angle(b.arm))
}

// The share of the drawing on the four branches from this centre, if they read as a swastika.
func readFrom(links [][]int, flats []drawings.Flat, w walk, centre int) (float64, bool) {
	ends := endsOf(links, flats, w, centre)
	if len(ends) != 4 {
		return 0, false
	}
	branches := make([]branch, 0, 4)
	onBranch := cpcolls.NewSet(centre)
	for _, end := range ends {
		path := w.pathTo(end)
		onBranch.Add(path...)
		b, ok := branchOf(path, flats)
		if !ok {
			return 0, false
		}
		branches = append(branches, b)
	}

	shortest, longest := branches[0].steps, branches[0].steps
	for _, b := range branches {
		shortest, longest = min(shortest, b.steps), max(longest, b.steps)
	}
	if float64(longest) > maxUneven*float64(shortest) {
		return 0, false
	}

	shortArms, left, right := 0, 0, 0
	arms := make([]float64, 0, 4)
	for _, b := range branches {
		// one step of arm and one of hook at least: the corner is neither the centre nor the tip
		if b.corner < 1 || b.corner > b.steps-1 {
			return 0, false
		}
		if b.corner == 1 {
			shortArms++
		}
		turn := degrees(b.turn())
		if math.Abs(turn) < minTurn || math.Abs(turn) > maxTurn {
			return 0, false
		}
		if turn > 0 {
			left++
		} else if turn < 0 {
			right++
		}
		arms = append(arms, math.Mod(angle(b.arm)+2*math.Pi, 2*math.Pi))
	}
	if shortArms > maxShortArms || (left != 4 && right != 4) {
		return 0, false
	}

	slices.Sort(arms)
	for i, arm := range arms {
		next := arms[(i+1)%4]
		if i == 3 {
			next += 2 * math.Pi
		}
		if gap := degrees(next - arm); gap < minGap || gap > maxGap {
			return 0, false
		}
	}

	near := cpcolls.NewSet[int]()
	onBranch.ForEach(func(tile int) {
		near.Add(tile)
		near.Add(links[tile]...)
	})
	n := float64(len(links))
	on := float64(onBranch.Len()) / n
	if float64(near.Len())/n < minCover || on < minOn {
		return 0, false
	}
	return on, true
}

// The ends of the strokes, two steps out at least. An end is a group of touching tiles as far from the centre as each
// other, with nothing farther next to any of them. On a hex grid both a one-tile hook at 60 degrees and the corner
// where a zigzag meets a straight stroke make such a pair; the corner's stroke goes on past it, the hook's does not.
// Of an end's tiles, the one whose path turns the most stands for it: of an arm's last tile and a one-tile hook as far
// from the centre, the hook.
func endsOf(links [][]int, flats []drawings.Flat, w walk, centre int) []int {
	order := slices.Clone(w.order)
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(w.steps[b], w.steps[a]) })

	var ends []int
	seen := make([]bool, len(links))
	for _, tile := range order {
		if seen[tile] || w.steps[tile] < 2 {
			continue
		}
		plateau := []int{tile}
		seen[tile] = true
		for next := 0; next < len(plateau); next++ {
			for _, neighbour := range links[plateau[next]] {
				if !seen[neighbour] && w.steps[neighbour] == w.steps[tile] {
					seen[neighbour] = true
					plateau = append(plateau, neighbour)
				}
			}
		}
		if goesOn(links, w, plateau) {
			continue
		}
		slices.Sort(plateau)
		ends = append(ends, slices.MaxFunc(plateau, func(a, b int) int {
			if c := cmp.Compare(turnTo(w, flats, a), turnTo(w, flats, b)); c != 0 {
				return c
			}
			return cmp.Compare(distance(flats[a], flats[centre]), distance(flats[b], flats[centre]))
		}))
	}
	return ends
}

func goesOn(links [][]int, w walk, plateau []int) bool {
	for _, tile := range plateau {
		for _, neighbour := range links[tile] {
			if w.steps[neighbour] > w.steps[tile] {
				return true
			}
		}
	}
	return false
}

func turnTo(w walk, flats []drawings.Flat, tile int) float64 {
	b, ok := branchOf(w.pathTo(tile), flats)
	if !ok {
		return 0
	}
	return math.Abs(b.turn())
}

func angle(v drawings.Flat) float64 {
	return math.Atan2(v.Y, v.X)
}

// Into (-pi, pi].
func wrap(a float64) float64 {
	a = math.Mod(a+math.Pi, 2*math.Pi)
	if a < 0 {
		a += 2 * math.Pi
	}
	return a - math.Pi
}

func degrees(radians float64) float64 {
	return radians * 180 / math.Pi
}

func distance(a, b drawings.Flat) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}
