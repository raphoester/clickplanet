package garrisonmessage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/garrisons"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/garrisonmessage"
)

func TestAGarrisonGoesOutWhole(t *testing.T) {
	garrison := garrisonmessage.Encode(garrisons.Garrison{Tile: 7, Country: "fr", Defenders: 3})

	assert.Equal(t, uint32(7), garrison.GetTileId())
	assert.Equal(t, "fr", garrison.GetCountryId())
	assert.Equal(t, uint32(3), garrison.GetDefenders())
}

func TestAGarrisonWithNoDefenderLeftSaysSo(t *testing.T) {
	garrison := garrisonmessage.Encode(garrisons.Garrison{Tile: 7, Country: "fr"})

	assert.Zero(t, garrison.GetDefenders())
	assert.Equal(t, "fr", garrison.GetCountryId())
}
