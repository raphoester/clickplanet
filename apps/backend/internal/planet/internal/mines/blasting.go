package mines

import "github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"

func NewBlasting(rules func() charges.Rules) Blasting {
	return Blasting{rules: rules}
}

// Blasting is how wide a mine's blast is under the rules in force, read at each explosion.
type Blasting struct {
	rules func() charges.Rules
}

func (b Blasting) Rings() int {
	return b.rules().Mine.Rings
}
