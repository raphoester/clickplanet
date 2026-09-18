package chargesheld_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/chargesheld"
)

func TestEveryChargeHeldGoesOut(t *testing.T) {
	held := chargesheld.Encode(bonuses.Held{Bomb: true, Enclose: true, SpreadClicks: 5})

	assert.True(t, held.GetBomb())
	assert.True(t, held.GetEnclose())
	assert.Equal(t, uint32(5), held.GetSpreadClicksLeft())
}

func TestAnEmptyHandIsSaidToo(t *testing.T) {
	held := chargesheld.Encode(bonuses.Held{})

	assert.False(t, held.GetBomb())
	assert.False(t, held.GetEnclose())
	assert.Zero(t, held.GetSpreadClicksLeft())
}
