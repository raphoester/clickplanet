package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func TestCapitalsAreListedByCountry(t *testing.T) {
	capitals := clicks.NewCapitals(map[string]uint32{"fr": 7, "de": 3})

	assert.Equal(t, []clicks.Capital{{Country: "de", Tile: 3}, {Country: "fr", Tile: 7}}, capitals.All())
	assert.True(t, capitals.Is(7))
	assert.False(t, capitals.Is(4))
}

func TestOnlyAFlagHasACapitalToPlay(t *testing.T) {
	capitals := clicks.NewCapitals(map[string]uint32{"fr": 7, "nc": 3}).Flags(func(country string) bool {
		return country != "nc"
	})

	assert.Equal(t, []clicks.Capital{{Country: "fr", Tile: 7}}, capitals.All())
	assert.False(t, capitals.Is(3))
}
