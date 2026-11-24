package cpflags_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

func TestACodeIsTheShapeOfAFlagTheGameShipsWithOrOfOneOpenedSince(t *testing.T) {
	for _, code := range []string{"fr", "xb", "gb-eng", "r-k3p9x2ma0b"} {
		assert.True(t, cpflags.Codes{}.CheckCountry(code), code)
	}
	for _, code := range []string{"", "FR", "fra", "gb-", "r-", "r-K3P9X2MA0B", "r-k3p9x2ma0", "r-k3p9x2ma0bc", "Rennes"} {
		assert.False(t, cpflags.Codes{}.CheckCountry(code), code)
	}
}

func TestAnOpenedCodeIsAFlagsCodeTheGameDoesNotShip(t *testing.T) {
	assert.True(t, cpflags.IsOpenedCode("r-0000000001"))
	assert.True(t, cpflags.IsOpenedCode("xb"), "Brittany, sold since it left the build")
	assert.False(t, cpflags.IsOpenedCode("fr"))
	assert.False(t, cpflags.IsOpenedCode("gb-eng"))
	assert.False(t, cpflags.IsOpenedCode("Rennes"))
}
