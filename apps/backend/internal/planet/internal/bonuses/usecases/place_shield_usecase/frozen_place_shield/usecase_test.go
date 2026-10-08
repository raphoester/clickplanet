package frozen_place_shield_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/place_shield_usecase/frozen_place_shield"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

type countingPlacement struct{ calls int }

func (c *countingPlacement) Execute(context.Context, place_shield_usecase.In) (bonuses.Held, error) {
	c.calls++
	return bonuses.Held{}, nil
}

func TestAShieldOnAFrozenMapIsRefusedAndKept(t *testing.T) {
	rules, err := tempo.NewRules(1, 0, true)
	require.NoError(t, err)
	switches := tempo.NewSwitches()
	switches.Set(rules)
	inner := &countingPlacement{}

	_, err = frozen_place_shield.New(inner, switches).Execute(t.Context(), place_shield_usecase.In{TileID: 1, CountryID: "fr"})

	require.ErrorIs(t, err, tempo.ErrFrozen)
	assert.Zero(t, inner.calls, "the shield is never spent")
}

func TestAShieldOnALiveMapIsPlaced(t *testing.T) {
	inner := &countingPlacement{}

	_, err := frozen_place_shield.New(inner, tempo.NewSwitches()).Execute(t.Context(), place_shield_usecase.In{TileID: 1, CountryID: "fr"})

	require.NoError(t, err)
	assert.Equal(t, 1, inner.calls)
}
