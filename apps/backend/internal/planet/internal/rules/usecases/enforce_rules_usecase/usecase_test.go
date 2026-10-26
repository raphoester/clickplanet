package enforce_rules_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/enforce_rules_usecase"
)

type enforced struct{ sets []rules.Snapshot }

func (e *enforced) Enforce(set rules.Set) { e.sets = append(e.sets, set.TakeSnapshot()) }

var plain = rules.TestSnapshot(rules.Default)

func keep(t *testing.T, store *rules.MemoryStore, kept rules.Snapshot) {
	t.Helper()

	set, err := kept.Restore()
	require.NoError(t, err)
	require.NoError(t, store.Save(t.Context(), set))
}

func TestTheFirstPassEnforcesTheDefaultTheDatabaseHolds(t *testing.T) {
	store, inForce := rules.NewMemoryStore(), &enforced{}
	keep(t, store, plain)

	played, err := enforce_rules_usecase.New(store, inForce).Execute(t.Context())

	require.NoError(t, err)
	assert.Equal(t, plain, played)
	assert.Equal(t, []rules.Snapshot{plain}, inForce.sets)
}

func TestASetUnchangedIsNotEnforcedAgainAndAnEditIs(t *testing.T) {
	store, inForce := rules.NewMemoryStore(), &enforced{}
	keep(t, store, plain)
	useCase := enforce_rules_usecase.New(store, inForce)
	_, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	again, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.Equal(t, rules.Snapshot{}, again)

	quicker := rules.TestSnapshot(rules.Default)
	quicker.Bonuses.Kinds[bonuses.KindBomb] = 4
	keep(t, store, quicker)
	edited, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.Equal(t, quicker, edited)
	assert.Equal(t, []rules.Snapshot{plain, quicker}, inForce.sets)
}

func TestNoDefaultKeptIsAnErrorAndEnforcesNothing(t *testing.T) {
	inForce := &enforced{}

	_, err := enforce_rules_usecase.New(rules.NewMemoryStore(), inForce).Execute(t.Context())

	require.ErrorIs(t, err, rules.ErrUnknownSet)
	assert.Empty(t, inForce.sets)
}

func TestAStoreThatCannotBeReadKeepsTheRulesInForce(t *testing.T) {
	store, inForce := rules.NewMemoryStore(), &enforced{}
	keep(t, store, plain)
	useCase := enforce_rules_usecase.New(store, inForce)
	_, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	store.FailWith(errors.New("postgres is down"))
	_, err = useCase.Execute(t.Context())

	require.Error(t, err)
	assert.Equal(t, []rules.Snapshot{plain}, inForce.sets)
}
