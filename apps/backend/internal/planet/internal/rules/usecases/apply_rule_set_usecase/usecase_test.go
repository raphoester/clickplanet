package apply_rule_set_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/apply_rule_set_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func setup(t *testing.T) (*apply_rule_set_usecase.UseCase, *rules.MemoryLayerStore, *cptime.FixedClock) {
	t.Helper()

	sets := rules.NewMemoryStore()
	for _, name := range []rules.SetName{rules.Default, "rush"} {
		set, err := rules.Snapshot{
			Name: name, Tempo: 1,
			Boxes: bonuses.Pace{MinInterval: time.Minute, MaxInterval: time.Minute, MaxChargesPerHour: 12},
		}.Restore()
		require.NoError(t, err)
		require.NoError(t, sets.Save(t.Context(), set))
	}

	layers, clock := rules.NewMemoryLayerStore(), cptime.NewFixedClock(at)
	return apply_rule_set_usecase.New(sets, layers, clock), layers, clock
}

func latest(t *testing.T, layers *rules.MemoryLayerStore) rules.BaselineSnapshot {
	t.Helper()

	baseline, err := layers.LatestBaseline(t.Context())
	require.NoError(t, err)
	return baseline.TakeSnapshot()
}

func TestARequestIsKeptAsTheBaseline(t *testing.T) {
	useCase, layers, _ := setup(t)

	require.NoError(t, useCase.Execute(t.Context(), apply_rule_set_usecase.In{Request: "season-0:finale", Set: "rush"}))

	assert.Equal(t, rules.BaselineSnapshot{Request: "season-0:finale", Set: "rush", AppliedAt: at}, latest(t, layers))
}

func TestTheSameRequestAgainChangesNothingAndIsNoError(t *testing.T) {
	useCase, layers, clock := setup(t)
	finale := apply_rule_set_usecase.In{Request: "season-0:finale", Set: "rush"}
	require.NoError(t, useCase.Execute(t.Context(), finale))
	clock.Advance(time.Hour)
	require.NoError(t, useCase.Execute(t.Context(), apply_rule_set_usecase.In{Request: "season-0:over", Set: rules.Default}))

	clock.Advance(time.Hour)
	require.NoError(t, useCase.Execute(t.Context(), finale), "a boot sends its phase again")

	assert.Equal(t, rules.BaselineSnapshot{Request: "season-0:over", Set: rules.Default, AppliedAt: at.Add(time.Hour)},
		latest(t, layers), "the request applied since stands")
}

func TestAnUnknownSetIsRefusedAndKeepsNothing(t *testing.T) {
	useCase, layers, _ := setup(t)

	err := useCase.Execute(t.Context(), apply_rule_set_usecase.In{Request: "r", Set: "frozen"})

	require.ErrorIs(t, err, rules.ErrUnknownSet)
	assert.Equal(t, rules.BaselineSnapshot{}, latest(t, layers))
}

func TestARequestWithNoIDIsRefused(t *testing.T) {
	useCase, layers, _ := setup(t)

	err := useCase.Execute(t.Context(), apply_rule_set_usecase.In{Set: "rush"})

	require.ErrorIs(t, err, rules.ErrNoRequest)
	assert.Equal(t, rules.BaselineSnapshot{}, latest(t, layers))
}

func TestAStoreThatCannotBeReadIsAnError(t *testing.T) {
	useCase, layers, _ := setup(t)
	down := errors.New("postgres is down")
	layers.FailWith(down)

	err := useCase.Execute(t.Context(), apply_rule_set_usecase.In{Request: "r", Set: "rush"})

	require.ErrorIs(t, err, down, "the next send tries again")
}
