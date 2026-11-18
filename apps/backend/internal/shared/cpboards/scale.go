package cpboards

import (
	"fmt"
	"math"
)

// Earth1Spacing is earth-1's spacing, the median arc between two touching tiles in radians (its manifest's): every
// threshold of the game was set on it. The page and the backoffice scale from the same number.
const Earth1Spacing = 0.004063093462255102

// Scale is how a board compares with earth-1: its spacing over earth-1's. On earth-1 every value comes out unchanged, and
// so does the zero Scale: earth-1's.
type Scale struct {
	length float64
}

func (s Scale) ratio() float64 {
	if s.length == 0 {
		return 1
	}
	return s.length
}

// Earth1 is earth-1's scale, which changes nothing: the zero Scale.
func Earth1() Scale {
	return Scale{}
}

// ScaleOf is the scale of the board of that name this app carries. An empty name is earth-1's: an event or an answer
// from before the boards were named.
func ScaleOf(name string) (Scale, error) {
	if name == "" {
		return Earth1(), nil
	}
	board, err := Named(Files(), name)
	if err != nil {
		return Scale{}, fmt.Errorf("failed to scale to board %q: %w", name, err)
	}
	return board.Scale(), nil
}

// Length is the board's spacing over earth-1's.
func (s Scale) Length() float64 {
	return s.ratio()
}

// Density is how many of the board's tiles cover the ground of one of earth-1's.
func (s Scale) Density() float64 {
	return 1 / (s.ratio() * s.ratio())
}

// Tiles is a count of earth-1's tiles on the board: as much ground, and one tile at least.
func (s Scale) Tiles(n int) int {
	return max(1, int(math.Round(float64(n)*s.Density())))
}

// Distance is a length on earth-1, as many tiles across on the board.
func (s Scale) Distance(x float64) float64 {
	return x * s.ratio()
}

// Zoom is a camera zoom on earth-1, where a tile is as wide on the screen on the board.
func (s Scale) Zoom(z float64) float64 {
	return z / s.ratio()
}
