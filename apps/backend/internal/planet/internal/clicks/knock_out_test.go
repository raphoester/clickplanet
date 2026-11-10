package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func TestATakeOfAStandingFlagsCapitalKnocksItOut(t *testing.T) {
	assert.NoError(t, clicks.KnockOutError(true, "fr", false, "de"))
}

func TestATakeKnocksNothingOut(t *testing.T) {
	cases := map[string]struct {
		arena     bool
		capitalOf string
		out       bool
		flag      string
	}{
		"on the world's map":           {arena: false, capitalOf: "fr", flag: "de"},
		"of a tile that is no capital": {arena: true, capitalOf: "", flag: "de"},
		"of a flag already out":        {arena: true, capitalOf: "fr", out: true, flag: "de"},
		"of the flag's own capital":    {arena: true, capitalOf: "fr", flag: "fr"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.ErrorIs(t, clicks.KnockOutError(c.arena, c.capitalOf, c.out, c.flag), clicks.ErrNoKnockOut)
		})
	}
}

func TestEveryFlagStandsOutsideAnArena(t *testing.T) {
	assert.True(t, clicks.Stands(false, false, false))
	assert.True(t, clicks.Stands(false, true, true))
}

func TestInAnArenaAFlagStandsWithACapitalNotOut(t *testing.T) {
	assert.True(t, clicks.Stands(true, true, false))
	assert.False(t, clicks.Stands(true, true, true), "out")
	assert.False(t, clicks.Stands(true, false, false), "a region flag, with no capital")
}
