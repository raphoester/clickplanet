package play_rules_usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/play_rules_usecase"
)

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

type enforced struct{ sets []rules.Snapshot }

func (e *enforced) Enforce(set rules.Set) { e.sets = append(e.sets, set.TakeSnapshot()) }

func (e *enforced) names() []rules.SetName {
	names := make([]rules.SetName, 0, len(e.sets))
	for _, set := range e.sets {
		names = append(names, set.Name)
	}
	return names
}

func keeping(t *testing.T, kept ...rules.Snapshot) *rules.MemoryStore {
	t.Helper()

	store := rules.NewMemoryStore(parts.All())
	for _, each := range kept {
		set, err := each.Restore(parts.All(), rules.Seeded())
		require.NoError(t, err)
		require.NoError(t, store.Save(t.Context(), set))
	}
	return store
}

var rush = rules.Named("rush", func(r *planetv1.Rules) { r.Clicks.Tempo = proto.Float64(3) })

func booted(t *testing.T, sets *rules.MemoryStore, choices *rules.MemoryChoices) (*play_rules_usecase.UseCase, *enforced) {
	t.Helper()

	inForce := &enforced{}
	useCase := play_rules_usecase.New(sets, choices, inForce)
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	return useCase, inForce
}

func TestAtFirstDefaultPlays(t *testing.T) {
	inForce := &enforced{}

	taken, took, err := play_rules_usecase.New(keeping(t, rules.Seeded()), rules.NewMemoryChoices(), inForce).
		Execute(t.Context())

	require.NoError(t, err)
	assert.True(t, took)
	assert.Equal(t, rules.Default, taken.Name)
	require.Len(t, inForce.sets, 1)
	assert.True(t, proto.Equal(rules.Seeded().Rules, inForce.sets[0].Rules))
}

func TestABootPlaysTheSetChosenLast(t *testing.T) {
	choices := rules.NewMemoryChoices()
	require.NoError(t, choices.Save(t.Context(), rules.Choice{Set: "rush", DecidedAt: finale}))

	_, inForce := booted(t, keeping(t, rules.Seeded(), rush), choices)

	assert.Equal(t, []rules.SetName{"rush"}, inForce.names())
}

func TestAChoicePlaysItsSetAndIsKept(t *testing.T) {
	choices := rules.NewMemoryChoices()
	useCase, inForce := booted(t, keeping(t, rules.Seeded(), rush), choices)

	taken, took, err := useCase.Play(t.Context(), rules.Choice{Set: "rush", DecidedAt: finale})

	require.NoError(t, err)
	assert.True(t, took)
	assert.Equal(t, rules.SetName("rush"), taken.Name)
	assert.Equal(t, []rules.SetName{rules.Default, "rush"}, inForce.names())
	kept, _, err := choices.Choice(t.Context())
	require.NoError(t, err)
	assert.Equal(t, rules.Choice{Set: "rush", DecidedAt: finale}, kept)
}

func TestTheLatestDecisionWinsAndOneSentAgainChangesNothing(t *testing.T) {
	useCase, inForce := booted(t, keeping(t, rules.Seeded(), rush), rules.NewMemoryChoices())
	over := rules.Choice{Set: rules.Default, DecidedAt: finale.Add(2 * time.Hour)}

	for _, choice := range []rules.Choice{
		{Set: "rush", DecidedAt: finale},
		over,
		{Set: "rush", DecidedAt: finale},
		over,
	} {
		_, _, err := useCase.Play(t.Context(), choice)
		require.NoError(t, err)
	}

	assert.Equal(t, []rules.SetName{rules.Default, "rush", rules.Default}, inForce.names())
}

func TestAChoiceOfTheSetInForceTakesNothing(t *testing.T) {
	useCase, inForce := booted(t, keeping(t, rules.Seeded()), rules.NewMemoryChoices())

	_, took, err := useCase.Play(t.Context(), rules.Choice{Set: rules.Default, DecidedAt: finale})

	require.NoError(t, err)
	assert.False(t, took)
	assert.Len(t, inForce.sets, 1)
}

func TestASetPlanetDoesNotKeepIsRefusedAndTheRulesInForceStay(t *testing.T) {
	choices := rules.NewMemoryChoices()
	useCase, inForce := booted(t, keeping(t, rules.Seeded()), choices)

	_, took, err := useCase.Play(t.Context(), rules.Choice{Set: "rush", DecidedAt: finale})

	require.ErrorIs(t, err, rules.ErrUnknownSet)
	assert.False(t, took)
	assert.Equal(t, []rules.SetName{rules.Default}, inForce.names())
	_, kept, err := choices.Choice(t.Context())
	require.NoError(t, err)
	assert.False(t, kept, "a refused choice is not kept")
}

func TestAKeptSetThatCannotPlayIsRefused(t *testing.T) {
	sets := keeping(t, rules.Seeded())
	sets.Keep(rules.Named("rush", func(r *planetv1.Rules) { r.Clicks.Burst = proto.Uint32(0) }))
	useCase, inForce := booted(t, sets, rules.NewMemoryChoices())

	_, _, err := useCase.Play(t.Context(), rules.Choice{Set: "rush", DecidedAt: finale})

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Len(t, inForce.sets, 1)
}

func TestAChoiceThatCannotBeKeptPlaysNothing(t *testing.T) {
	choices := rules.NewMemoryChoices()
	useCase, inForce := booted(t, keeping(t, rules.Seeded(), rush), choices)

	choices.FailWith(errors.New("postgres is down"))
	_, _, err := useCase.Play(t.Context(), rules.Choice{Set: "rush", DecidedAt: finale})

	require.Error(t, err)
	assert.Len(t, inForce.sets, 1)

	choices.FailWith(nil)
	_, took, err := useCase.Play(t.Context(), rules.Choice{Set: "rush", DecidedAt: finale})
	require.NoError(t, err)
	assert.True(t, took, "the command delivered again plays it")
}

func TestWithNoDefaultNothingPlays(t *testing.T) {
	inForce := &enforced{}

	_, _, err := play_rules_usecase.New(rules.NewMemoryStore(parts.All()), rules.NewMemoryChoices(), inForce).
		Execute(t.Context())

	require.ErrorIs(t, err, rules.ErrUnknownSet)
	assert.Empty(t, inForce.sets)
}

func TestAChangeOfTheSetInForceIsTakenAtTheNextExecute(t *testing.T) {
	sets := keeping(t, rules.Seeded())
	useCase, inForce := booted(t, sets, rules.NewMemoryChoices())

	_, took, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.False(t, took, "unchanged, it is not taken again")

	sets.Keep(rules.Named(rules.Default, func(r *planetv1.Rules) { r.Clicks.NewAccountClicks = proto.Float64(60) }))
	_, took, err = useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.True(t, took)
	require.Len(t, inForce.sets, 2)
	assert.InDelta(t, 60, inForce.sets[1].Rules.GetClicks().GetNewAccountClicks(), 0)
}
