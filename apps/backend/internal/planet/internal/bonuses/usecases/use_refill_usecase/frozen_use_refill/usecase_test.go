package frozen_use_refill_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/use_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/use_refill_usecase/frozen_use_refill"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

type countingRefill struct{ calls int }

func (c *countingRefill) Execute(context.Context, use_refill_usecase.In) (use_refill_usecase.Out, error) {
	c.calls++
	return use_refill_usecase.Out{}, nil
}

func TestARefillOnAFrozenMapIsRefusedAndKept(t *testing.T) {
	rules, err := tempo.NewRules(1, 0, true)
	require.NoError(t, err)
	switches := tempo.NewSwitches()
	switches.Set(rules)
	inner := &countingRefill{}

	_, err = frozen_use_refill.New(inner, switches).Execute(t.Context(), use_refill_usecase.In{CountryID: "fr"})

	require.ErrorIs(t, err, tempo.ErrFrozen)
	assert.Zero(t, inner.calls, "the refill is never spent")
}

func TestARefillOnALiveMapIsUsed(t *testing.T) {
	inner := &countingRefill{}

	_, err := frozen_use_refill.New(inner, tempo.NewSwitches()).Execute(t.Context(), use_refill_usecase.In{CountryID: "fr"})

	require.NoError(t, err)
	assert.Equal(t, 1, inner.calls)
}
