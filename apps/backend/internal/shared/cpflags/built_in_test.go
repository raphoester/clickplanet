package cpflags_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

func TestTheGameShipsWithThePagesFlagsInItsOrder(t *testing.T) {
	flags := cpflags.BuiltIn()

	require.Len(t, flags, 235)
	assert.Equal(t, cpflags.BuiltInFlag{Code: "af", Name: "Afghanistan"}, flags[0])
	assert.Contains(t, flags, cpflags.BuiltInFlag{Code: "gb", Name: "UK"})
	for _, sold := range []string{"xb", "gb-eng", "gb-sct", "gb-wls"} {
		assert.False(t, cpflags.IsBuiltIn(sold), "%s left the build: it is sold", sold)
	}
}

func TestNoFlagTheGameShipsWithLooksLikeOneOpenedSince(t *testing.T) {
	shipped := regexp.MustCompile(`^[a-z]{2}(-[a-z]{3})?$`)
	for _, flag := range cpflags.BuiltIn() {
		assert.Regexp(t, shipped, flag.Code)
		assert.True(t, cpflags.IsBuiltIn(flag.Code))
	}
	assert.False(t, cpflags.IsBuiltIn("r-k3p9x2ma0b"))
	assert.False(t, cpflags.IsBuiltIn(""))
}
