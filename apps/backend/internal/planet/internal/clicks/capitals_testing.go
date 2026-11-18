//go:build testing

package clicks

// NoCapitals is a map with no capital, and no capital rule in force.
func NoCapitals() CapitalPlay {
	return NewCapitalPlay(func() *Capitals { return NewCapitals(map[string]uint32{}) }, func() CapitalRules { return CapitalRules{} })
}
