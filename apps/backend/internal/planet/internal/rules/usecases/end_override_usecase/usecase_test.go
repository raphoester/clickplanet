package end_override_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/end_override_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

var rushForAnHour = rules.OverrideSnapshot{Set: "rush", StartsAt: at, EndsAt: at.Add(time.Hour)}

func setup(t *testing.T, override rules.OverrideSnapshot) (*end_override_usecase.UseCase, *rules.MemoryLayerStore, *cptime.FixedClock) {
	t.Helper()

	layers, clock := rules.NewMemoryLayerStore(), cptime.NewFixedClock(at)
	if override != (rules.OverrideSnapshot{}) {
		require.NoError(t, layers.SaveOverride(t.Context(), override.Restore()))
	}
	return end_override_usecase.New(layers, clock), layers, clock
}

func kept(t *testing.T, layers *rules.MemoryLayerStore) rules.OverrideSnapshot {
	t.Helper()

	override, err := layers.Override(t.Context())
	require.NoError(t, err)
	return override.TakeSnapshot()
}

func TestTheOverrideRunningEndsNow(t *testing.T) {
	useCase, layers, clock := setup(t, rushForAnHour)
	clock.Advance(10 * time.Minute)

	require.NoError(t, useCase.Execute(t.Context()))

	assert.Equal(t, rules.OverrideSnapshot{Set: "rush", StartsAt: at, EndsAt: at.Add(10 * time.Minute)}, kept(t, layers))
}

func TestWithNoOverrideRunningNothingChanges(t *testing.T) {
	useCase, layers, clock := setup(t, rushForAnHour)
	clock.Advance(2 * time.Hour)

	require.NoError(t, useCase.Execute(t.Context()))
	assert.Equal(t, rushForAnHour, kept(t, layers), "its end stays the one it had")

	none, empty, _ := setup(t, rules.OverrideSnapshot{})
	require.NoError(t, none.Execute(t.Context()))
	assert.Equal(t, rules.OverrideSnapshot{}, kept(t, empty), "no override is made to end one")
}

func TestAnOverrideThatCannotBeReadIsAnError(t *testing.T) {
	useCase, layers, _ := setup(t, rushForAnHour)
	down := errors.New("postgres is down")
	layers.FailWith(down)

	require.ErrorIs(t, useCase.Execute(t.Context()), down)
}
