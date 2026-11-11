package clicks

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// CapitalRules is what a rule set says of the capitals. KnockOut: a take of a flag's capital by another flag clears
// every other tile of the flag, and the flag is out. FixedShields: no shield is placed on a capital, and a fortify skips
// it.
type CapitalRules struct {
	KnockOut     bool
	FixedShields bool
}

func (CapitalRules) Validate() error { return nil }

var CapitalsPart = cprules.PartOf[*planetv1.Rules, CapitalRules]{
	Key: "capitals",
	Read: func(m *planetv1.Rules) CapitalRules {
		return CapitalRules{KnockOut: m.GetCapitals().GetKnockOut(), FixedShields: m.GetCapitals().GetFixedShields()}
	},
}

// CapitalPlay is the map's capitals as the rules in force play them.
type CapitalPlay struct {
	capitals *Capitals
	rules    func() CapitalRules
}

func NewCapitalPlay(capitals *Capitals, rules func() CapitalRules) CapitalPlay {
	return CapitalPlay{capitals: capitals, rules: rules}
}

// Fixes is true for a capital no shield is added to.
func (p CapitalPlay) Fixes(tile uint32) bool {
	return p.rules().FixedShields && p.capitals.Is(tile)
}

// Felled is the flag a take of the tile by flag knocks out, given whether a flag is out already: ErrNoKnockOut when it
// knocks none.
func (p CapitalPlay) Felled(tile uint32, flag string, out func(flag string) bool) (string, error) {
	country := p.capitals.CountryOf(tile)
	if err := p.rules().KnockOutError(country, out(country), flag); err != nil {
		return "", err
	}
	return country, nil
}

// Stands says whether the flag is still in the game, given whether it is out.
func (p CapitalPlay) Stands(flag string, out bool) bool {
	return p.rules().Stands(p.capitals.Has(flag), out)
}
