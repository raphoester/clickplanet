package restore_rules_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/restore_rules_usecase"
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

func keep(t *testing.T, store *rules.MemoryStore, kept rules.Snapshot) {
	t.Helper()

	require.NoError(t, store.Save(t.Context(), kept.Restore()))
}

func TestABootEnforcesTheSetAppliedLast(t *testing.T) {
	store, inForce := rules.NewMemoryStore(), &enforced{}
	keep(t, store, rules.Snapshot{Request: "a", Set: rules.Default, Source: rules.SourceOperator, AppliedAt: at})
	keep(t, store, rules.Snapshot{Request: "season-0:finale", Set: "rush", Source: rules.SourceInternal, AppliedAt: at.Add(time.Minute)})

	require.NoError(t, restore_rules_usecase.New(store, catalog(t), inForce).Execute(t.Context()))

	assert.Equal(t, []rules.SetName{"rush"}, inForce.sets)
}

func TestABootWithNoChangeEnforcesTheDefaults(t *testing.T) {
	inForce := &enforced{}

	require.NoError(t, restore_rules_usecase.New(rules.NewMemoryStore(), catalog(t), inForce).Execute(t.Context()))

	assert.Equal(t, []rules.SetName{rules.Default}, inForce.sets)
}

func TestASetNoLongerConfiguredRefusesTheBoot(t *testing.T) {
	store, inForce := rules.NewMemoryStore(), &enforced{}
	keep(t, store, rules.Snapshot{Request: "a", Set: "gone", Source: rules.SourceOperator, AppliedAt: at})

	err := restore_rules_usecase.New(store, catalog(t), inForce).Execute(t.Context())

	require.ErrorIs(t, err, rules.ErrUnknownSet)
	assert.Empty(t, inForce.sets)
}

func TestAStoreThatCannotBeReadRefusesTheBoot(t *testing.T) {
	store := rules.NewMemoryStore()
	down := errors.New("postgres is down")
	store.FailWith(down)

	require.ErrorIs(t, restore_rules_usecase.New(store, catalog(t), &enforced{}).Execute(t.Context()), down)
}
