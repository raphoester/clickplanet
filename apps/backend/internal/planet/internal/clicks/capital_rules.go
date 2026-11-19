package clicks

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// CapitalRules is what a rule set says of the capitals. KnockOut: a take of a flag's capital by another flag clears
// every other tile of the flag, and the flag is out. FixedShields: no shield is placed on a capital, and a fortify skips
// it. CutOff: a take that cuts a flag's land from its capital clears the land cut off.
type CapitalRules struct {
	KnockOut     bool
	FixedShields bool
	CutOff       bool
}

func (CapitalRules) Validate() error { return nil }

var CapitalsPart = cprules.PartOf[*planetv1.Rules, CapitalRules]{
	Key: "capitals",
	Read: func(m *planetv1.Rules) CapitalRules {
		return CapitalRules{
			KnockOut: m.GetCapitals().GetKnockOut(), FixedShields: m.GetCapitals().GetFixedShields(),
			CutOff: m.GetCapitals().GetCutOff(),
		}
	},
}

// CapitalPlay is the capitals of the board in play as the rules in force play them.
type CapitalPlay struct {
	capitals func() *Capitals
	rules    func() CapitalRules
}

func NewCapitalPlay(capitals func() *Capitals, rules func() CapitalRules) CapitalPlay {
	return CapitalPlay{capitals: capitals, rules: rules}
}

// Fixes is true for a capital no shield is added to.
func (p CapitalPlay) Fixes(tile uint32) bool {
	return p.rules().FixedShields && p.capitals().Is(tile)
}

// KnocksOut is true for a capital whose take knocks its flag out under the rules in force.
func (p CapitalPlay) KnocksOut(tile uint32) bool {
	return p.rules().KnockOut && p.capitals().Is(tile)
}

// Felled is the flag a take of the tile by flag knocks out, given whether a flag is out already: ErrNoKnockOut when it
// knocks none.
func (p CapitalPlay) Felled(tile uint32, flag string, out func(flag string) bool) (string, error) {
	country := p.capitals().CountryOf(tile)
	if err := p.rules().KnockOutError(country, out(country), flag); err != nil {
		return "", err
	}
	return country, nil
}

// Stands says whether the flag is still in the game, given whether it is out.
func (p CapitalPlay) Stands(flag string, out bool) bool {
	return p.rules().Stands(p.capitals().Has(flag), out)
}

// Stranded is the flag's land that the loss of the tiles lost cut from its capital, read by holds after they left it:
// ErrNoCutOff when they cut none, or while the cut-off does not play.
func (p CapitalPlay) Stranded(flag string, lost []uint32, links Links, holds func(tile uint32) bool) ([]uint32, error) {
	capital, ok := p.capitals().Of(flag)
	if err := p.rules().CutOffError(ok); err != nil {
		return nil, err
	}
	cut := stranded(links, capital, holds, lost)
	if len(cut) == 0 {
		return nil, ErrNoCutOff
	}
	return cut, nil
}
