package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

func TestACountrysGroundIsEveryLandmassItHolds(t *testing.T) {
	borders, err := clicks.NewBorders("test", []uint16{1, 1, 2, 0, 3, 3, 3}, []string{"", "fr", "fr", "de"})
	require.NoError(t, err)

	assert.Equal(t, []clicks.Ground{{Country: "de", Tiles: 3}, {Country: "fr", Tiles: 3}}, borders.Grounds(),
		"the sea is no ground")
}
