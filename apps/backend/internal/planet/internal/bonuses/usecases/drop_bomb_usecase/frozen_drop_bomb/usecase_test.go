package frozen_drop_bomb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase/frozen_drop_bomb"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

type countingDrop struct{ calls int }

func (c *countingDrop) Execute(context.Context, drop_bomb_usecase.In) (clicks.Blast, error) {
	c.calls++
	return clicks.Blast{}, nil
}

func TestABombOnAFrozenMapIsRefusedAndKept(t *testing.T) {
	rules, err := tempo.NewRules(1, 0, true)
	require.NoError(t, err)
	switches := tempo.NewSwitches()
	switches.Set(rules)
	inner := &countingDrop{}

	_, err = frozen_drop_bomb.New(inner, switches).Execute(t.Context(), drop_bomb_usecase.In{CountryID: "fr"})

	require.ErrorIs(t, err, tempo.ErrFrozen)
	assert.Zero(t, inner.calls, "the bomb is never spent")
}

func TestABombOnALiveMapIsDropped(t *testing.T) {
	inner := &countingDrop{}

	_, err := frozen_drop_bomb.New(inner, tempo.NewSwitches()).Execute(t.Context(), drop_bomb_usecase.In{CountryID: "fr"})

	require.NoError(t, err)
	assert.Equal(t, 1, inner.calls)
}
