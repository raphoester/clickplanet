package override_rules_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/override_rules_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func setup(t *testing.T) (*override_rules_usecase.UseCase, *rules.MemoryLayerStore, *cptime.FixedClock) {
	t.Helper()

	sets := rules.NewMemoryStore()
	set, err := rules.Snapshot{
		Name: "rush", Tempo: 3,
		Boxes: bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30},
	}.Restore()
	require.NoError(t, err)
	require.NoError(t, sets.Save(t.Context(), set))

	layers, clock := rules.NewMemoryLayerStore(), cptime.NewFixedClock(at)
	return override_rules_usecase.New(sets, layers, clock), layers, clock
}

func kept(t *testing.T, layers *rules.MemoryLayerStore) rules.OverrideSnapshot {
	t.Helper()

	override, err := layers.Override(t.Context())
	require.NoError(t, err)
	return override.TakeSnapshot()
}

func TestAnOverrideRunsFromNowUntilTheEndAsked(t *testing.T) {
	useCase, layers, _ := setup(t)

	require.NoError(t, useCase.Execute(t.Context(), override_rules_usecase.In{Set: "rush", Until: at.Add(30 * time.Minute)}))

	assert.Equal(t, rules.OverrideSnapshot{Set: "rush", StartsAt: at, EndsAt: at.Add(30 * time.Minute)}, kept(t, layers))
}

func TestANewOverrideReplacesTheOneRunning(t *testing.T) {
	useCase, layers, clock := setup(t)
	require.NoError(t, useCase.Execute(t.Context(), override_rules_usecase.In{Set: "rush", Until: at.Add(time.Hour)}))
	clock.Advance(10 * time.Minute)

	require.NoError(t, useCase.Execute(t.Context(), override_rules_usecase.In{Set: "rush", Until: at.Add(20 * time.Minute)}))

	assert.Equal(t, rules.OverrideSnapshot{
		Set: "rush", StartsAt: at.Add(10 * time.Minute), EndsAt: at.Add(20 * time.Minute),
	}, kept(t, layers))
}

func TestAnOverrideOfAnUnknownSetOrWithNoEndAheadKeepsNothing(t *testing.T) {
	useCase, layers, _ := setup(t)

	err := useCase.Execute(t.Context(), override_rules_usecase.In{Set: "frozen", Until: at.Add(time.Hour)})
	require.ErrorIs(t, err, rules.ErrUnknownSet)

	err = useCase.Execute(t.Context(), override_rules_usecase.In{Set: "rush", Until: at})
	require.ErrorIs(t, err, rules.ErrInvalidOverride)

	assert.Equal(t, rules.OverrideSnapshot{}, kept(t, layers))
}

func TestAnOverrideThatCannotBeKeptIsAnError(t *testing.T) {
	useCase, layers, _ := setup(t)
	down := errors.New("postgres is down")
	layers.FailWith(down)

	err := useCase.Execute(t.Context(), override_rules_usecase.In{Set: "rush", Until: at.Add(time.Hour)})

	require.ErrorIs(t, err, down)
}
