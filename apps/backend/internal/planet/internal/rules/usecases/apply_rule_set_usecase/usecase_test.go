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

type enforced struct{ sets []rules.SetName }

func (e *enforced) Enforce(set rules.Set) { e.sets = append(e.sets, set.Name) }

func catalog(t *testing.T) rules.Catalog {
	t.Helper()

	catalog, err := rules.NewCatalog(rules.Config{Sets: map[rules.SetName]rules.SetConfig{"rush": {Tempo: 3}}},
		rules.Set{Name: rules.Default, Tempo: 1, Boxes: bonuses.Pace{MinInterval: time.Minute, MaxInterval: time.Minute}})
	require.NoError(t, err)
	return catalog
}

func setup(t *testing.T) (*apply_rule_set_usecase.UseCase, *rules.MemoryStore, *enforced, *cptime.FixedClock) {
	t.Helper()

	store, inForce, clock := rules.NewMemoryStore(), &enforced{}, cptime.NewFixedClock(at)
	return apply_rule_set_usecase.New(catalog(t), store, inForce, clock), store, inForce, clock
}

func TestARequestIsKeptThenEnforced(t *testing.T) {
	useCase, store, inForce, _ := setup(t)

	require.NoError(t, useCase.Execute(t.Context(), apply_rule_set_usecase.In{
		Request: "season-0:finale", Set: "rush", Source: rules.SourceInternal,
	}))

	assert.Equal(t, []rules.Snapshot{{Request: "season-0:finale", Set: "rush", Source: rules.SourceInternal, AppliedAt: at}},
		store.Kept())
	assert.Equal(t, []rules.SetName{"rush"}, inForce.sets)
}

func TestTheSameRequestAgainChangesNothingAndIsNoError(t *testing.T) {
	useCase, store, inForce, clock := setup(t)
	in := apply_rule_set_usecase.In{Request: "season-0:finale", Set: "rush", Source: rules.SourceInternal}
	require.NoError(t, useCase.Execute(t.Context(), in))
	require.NoError(t, useCase.Execute(t.Context(), apply_rule_set_usecase.In{
		Request: "a1b2", Set: rules.Default, Source: rules.SourceOperator,
	}))

	clock.Advance(time.Hour)
	require.NoError(t, useCase.Execute(t.Context(), in), "a boot sends its phase again")

	assert.Len(t, store.Kept(), 2)
	assert.Equal(t, []rules.SetName{"rush", rules.Default}, inForce.sets, "the operator's change stands")
}

func TestAnUnknownSetIsRefusedAndKeepsNothing(t *testing.T) {
	useCase, store, inForce, _ := setup(t)

	err := useCase.Execute(t.Context(), apply_rule_set_usecase.In{Request: "r", Set: "frozen", Source: rules.SourceOperator})

	require.ErrorIs(t, err, rules.ErrUnknownSet)
	assert.Empty(t, store.Kept())
	assert.Empty(t, inForce.sets)
}

func TestARequestWithNoIDIsRefused(t *testing.T) {
	useCase, store, _, _ := setup(t)

	err := useCase.Execute(t.Context(), apply_rule_set_usecase.In{Set: "rush", Source: rules.SourceOperator})

	require.ErrorIs(t, err, rules.ErrNoRequest)
	assert.Empty(t, store.Kept())
}

func TestAChangeThatCouldNotBeKeptIsNotEnforced(t *testing.T) {
	useCase, store, inForce, _ := setup(t)
	down := errors.New("postgres is down")
	store.FailWith(down)

	err := useCase.Execute(t.Context(), apply_rule_set_usecase.In{Request: "r", Set: "rush", Source: rules.SourceInternal})

	require.ErrorIs(t, err, down)
	assert.Empty(t, inForce.sets, "the next send tries again")
}
