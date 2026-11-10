package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var knockingOut = clicks.CapitalRules{KnockOut: true}

func TestATakeOfAStandingFlagsCapitalKnocksItOut(t *testing.T) {
	assert.NoError(t, knockingOut.KnockOutError("fr", false, "de"))
}

func TestATakeKnocksNothingOut(t *testing.T) {
	cases := map[string]struct {
		rules     clicks.CapitalRules
		capitalOf string
		out       bool
		flag      string
	}{
		"while the knock-out does not play": {rules: clicks.CapitalRules{}, capitalOf: "fr", flag: "de"},
		"of a tile that is no capital":      {rules: knockingOut, capitalOf: "", flag: "de"},
		"of a flag already out":             {rules: knockingOut, capitalOf: "fr", out: true, flag: "de"},
		"of the flag's own capital":         {rules: knockingOut, capitalOf: "fr", flag: "fr"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.ErrorIs(t, c.rules.KnockOutError(c.capitalOf, c.out, c.flag), clicks.ErrNoKnockOut)
		})
	}
}

func TestEveryFlagStandsWhileTheKnockOutDoesNotPlay(t *testing.T) {
	assert.True(t, clicks.CapitalRules{}.Stands(false, false))
	assert.True(t, clicks.CapitalRules{}.Stands(true, true))
}

func TestWhileTheKnockOutPlaysAFlagStandsWithACapitalNotOut(t *testing.T) {
	assert.True(t, knockingOut.Stands(true, false))
	assert.False(t, knockingOut.Stands(true, true), "out")
	assert.False(t, knockingOut.Stands(false, false), "a region flag, with no capital")
}

func TestACapitalsShieldsAreFixedOnlyWhileTheRuleSaysSo(t *testing.T) {
	capitals := clicks.NewCapitals(map[string]uint32{"fr": 7})
	fixed := clicks.CapitalRules{FixedShields: true}

	assert.True(t, clicks.NewCapitalPlay(capitals, func() clicks.CapitalRules { return fixed }).Fixes(7))
	assert.False(t, clicks.NewCapitalPlay(capitals, func() clicks.CapitalRules { return fixed }).Fixes(8), "no capital")
	assert.False(t, clicks.NewCapitalPlay(capitals, func() clicks.CapitalRules { return clicks.CapitalRules{} }).Fixes(7))
}
