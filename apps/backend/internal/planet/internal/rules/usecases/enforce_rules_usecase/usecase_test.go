package enforce_rules_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/enforce_rules_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var at = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

type enforced struct{ sets []rules.Snapshot }

func (e *enforced) Enforce(set rules.Set) { e.sets = append(e.sets, set.TakeSnapshot()) }

var plain = rules.Snapshot{
	Name:  rules.Default,
	Tempo: 1,
	Boxes: bonuses.Pace{MinInterval: 4 * time.Minute, MaxInterval: 8 * time.Minute, MaxChargesPerHour: 12},
	Crowd: 6,
}

func keep(t *testing.T, store *rules.MemoryStore, kept rules.Snapshot) {
	t.Helper()

	set, err := kept.Restore()
	require.NoError(t, err)
	require.NoError(t, store.Save(t.Context(), set))
}

func TestWithNoLayerTheFirstPassEnforcesTheDefault(t *testing.T) {
	store, inForce := rules.NewMemoryStore(), &enforced{}
	keep(t, store, plain)

	played, err := enforce_rules_usecase.New(store, rules.NewMemoryLayerStore(), inForce, cptime.NewFixedClock(at)).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, plain, played)
	assert.Equal(t, []rules.Snapshot{plain}, inForce.sets)
}

func TestASetUnchangedIsNotEnforcedAgainAndAnEditIs(t *testing.T) {
	store, inForce := rules.NewMemoryStore(), &enforced{}
	keep(t, store, plain)
	useCase := enforce_rules_usecase.New(store, rules.NewMemoryLayerStore(), inForce, cptime.NewFixedClock(at))
	_, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	again, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, rules.Snapshot{}, again)

	quicker := plain
	quicker.Tempo = 2
	keep(t, store, quicker)
	edited, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, quicker, edited)
	assert.Equal(t, []rules.Snapshot{plain, quicker}, inForce.sets)
}

func TestNoDefaultKeptIsAnErrorAndEnforcesNothing(t *testing.T) {
	inForce := &enforced{}

	_, err := enforce_rules_usecase.New(rules.NewMemoryStore(), rules.NewMemoryLayerStore(), inForce, cptime.NewFixedClock(at)).Execute(t.Context())

	require.ErrorIs(t, err, rules.ErrUnknownSet)
	assert.Empty(t, inForce.sets)
}

func TestAStoreThatCannotBeReadKeepsTheRulesInForce(t *testing.T) {
	store, inForce := rules.NewMemoryStore(), &enforced{}
	keep(t, store, plain)
	useCase := enforce_rules_usecase.New(store, rules.NewMemoryLayerStore(), inForce, cptime.NewFixedClock(at))
	_, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	store.FailWith(errors.New("postgres is down"))
	_, err = useCase.Execute(t.Context())

	require.Error(t, err)
	assert.Equal(t, []rules.Snapshot{plain}, inForce.sets)
}

var rush = rules.Snapshot{
	Name:  "rush",
	Tempo: 3,
	Boxes: bonuses.Pace{MinInterval: time.Minute, MaxInterval: 2 * time.Minute, MaxChargesPerHour: 30},
	Crowd: 50,
}

func TestTheBaselineIsEnforcedThenAnOverrideOverItUntilItsEnd(t *testing.T) {
	store, layers, inForce, clock := rules.NewMemoryStore(), rules.NewMemoryLayerStore(), &enforced{},
		cptime.NewFixedClock(at)
	keep(t, store, plain)
	keep(t, store, rush)
	useCase := enforce_rules_usecase.New(store, layers, inForce, clock)
	finale := rules.BaselineSnapshot{Request: "season-0:finale", Set: "rush", AppliedAt: at}
	require.NoError(t, layers.SaveBaseline(t.Context(), finale.Restore()))

	played, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, rush, played)

	calm := rules.OverrideSnapshot{Set: rules.Default, StartsAt: at, EndsAt: at.Add(time.Minute)}
	require.NoError(t, layers.SaveOverride(t.Context(), calm.Restore()))
	played, err = useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, plain, played)

	clock.Advance(time.Minute)
	played, err = useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, rush, played, "the baseline comes back once the override ends")

	assert.Equal(t, []rules.Snapshot{rush, plain, rush}, inForce.sets)
}

func TestLayersThatCannotBeReadKeepTheRulesInForce(t *testing.T) {
	store, layers, inForce := rules.NewMemoryStore(), rules.NewMemoryLayerStore(), &enforced{}
	keep(t, store, plain)
	useCase := enforce_rules_usecase.New(store, layers, inForce, cptime.NewFixedClock(at))
	_, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	layers.FailWith(errors.New("postgres is down"))
	_, err = useCase.Execute(t.Context())

	require.Error(t, err)
	assert.Equal(t, []rules.Snapshot{plain}, inForce.sets)
}
