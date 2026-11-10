//go:build testing

package clicks

// NoCapitals is a map with no capital, and no capital rule in force.
func NoCapitals() CapitalPlay {
	return NewCapitalPlay(NewCapitals(map[string]uint32{}), func() CapitalRules { return CapitalRules{} })
}

// CapitalsPlayed is the capitals, with every capital rule in force.
func CapitalsPlayed(capitals *Capitals) CapitalPlay {
	return NewCapitalPlay(capitals, func() CapitalRules { return CapitalRules{KnockOut: true, FixedShields: true} })
}
