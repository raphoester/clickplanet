//go:build testing

package clicks

// Spacing is a board's spacing, fixed: what a test of a bomb or a rally measures with.
type Spacing float64

func (s Spacing) Spacing() float64 {
	return float64(s)
}
