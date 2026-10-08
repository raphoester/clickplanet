package frozen_click_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase/frozen_click"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo"
)

type countingClick struct{ calls int }

func (c *countingClick) Execute(context.Context, click_usecase.In) (click_usecase.Out, error) {
	c.calls++
	return click_usecase.Out{}, nil
}

func frozen(t *testing.T) *tempo.Switches {
	t.Helper()

	rules, err := tempo.NewRules(1, 0, true)
	require.NoError(t, err)
	switches := tempo.NewSwitches()
	switches.Set(rules)
	return switches
}

func TestAClickOnAFrozenMapIsRefusedBeforeAnythingElseRuns(t *testing.T) {
	inner := &countingClick{}

	_, err := frozen_click.New(inner, frozen(t)).Execute(t.Context(), click_usecase.In{TileID: 1, CountryID: "fr"})

	require.ErrorIs(t, err, tempo.ErrFrozen)
	assert.Zero(t, inner.calls, "no token is spent and no try is timed")
}

func TestAClickOnALiveMapGoesThrough(t *testing.T) {
	inner := &countingClick{}

	_, err := frozen_click.New(inner, tempo.NewSwitches()).Execute(t.Context(), click_usecase.In{TileID: 1, CountryID: "fr"})

	require.NoError(t, err)
	assert.Equal(t, 1, inner.calls)
}
