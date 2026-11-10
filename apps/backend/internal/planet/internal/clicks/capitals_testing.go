//go:build testing

package clicks

// NoCapitals is a map whose arenas start blank.
func NoCapitals() *Capitals {
	return NewCapitals(map[string]uint32{})
}
