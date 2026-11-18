package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var knockingOut = clicks.CapitalRules{KnockOut: true}

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
