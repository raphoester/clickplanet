package cprules_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

type enforced struct{ sets []snapshot }

func (e *enforced) Enforce(taken set) { e.sets = append(e.sets, taken.TakeSnapshot()) }

func (e *enforced) names() []cprules.SetName {
	names := make([]cprules.SetName, 0, len(e.sets))
	for _, each := range e.sets {
		names = append(names, each.Name)
	}
	return names
}

// told is a hook: the antibot's tells planet its running bans again.
type told struct {
	mu   sync.Mutex
	sets []cprules.SetName
	err  error
}

func (t *told) hook(_ context.Context, taken cprules.SetName) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.sets = append(t.sets, taken)
	return t.err
}

func (t *told) heard() []cprules.SetName {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.sets
}

func keeping(kept ...snapshot) *cprules.MemoryStore[*planetv1.Rules] {
	return cprules.NewMemoryStore(parts, kept...)
}

func booted(
	t *testing.T, sets *cprules.MemoryStore[*planetv1.Rules], choices *cprules.MemoryChoices,
) (*cprules.PlayUseCase[*planetv1.Rules], *enforced) {
	t.Helper()

	inForce := &enforced{}
	useCase := cprules.NewPlayUseCase(sets, choices, inForce, nil)
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	return useCase, inForce
}

func TestAtFirstDefaultPlays(t *testing.T) {
	inForce := &enforced{}

	taken, took, err := cprules.NewPlayUseCase(keeping(seeded()), cprules.NewMemoryChoices(), inForce, nil).
		Execute(t.Context())

	require.NoError(t, err)
	assert.True(t, took)
	assert.Equal(t, cprules.Default, taken.Name)
	require.Len(t, inForce.sets, 1)
	assert.True(t, proto.Equal(seeded().Rules, inForce.sets[0].Rules))
}

func TestABootPlaysTheSetChosenLast(t *testing.T) {
	choices := cprules.NewMemoryChoices()
	require.NoError(t, choices.Save(t.Context(), cprules.Choice{Set: "rush", DecidedAt: finale}))

	_, inForce := booted(t, keeping(seeded(), rush()), choices)

	assert.Equal(t, []cprules.SetName{"rush"}, inForce.names())
	assert.True(t, proto.Equal(rush().Rules, inForce.sets[0].Rules))
}

func TestAChoicePlaysItsSetAndIsKept(t *testing.T) {
	choices := cprules.NewMemoryChoices()
	useCase, inForce := booted(t, keeping(seeded(), rush()), choices)

	taken, took, err := useCase.Play(t.Context(), cprules.Choice{Set: "rush", DecidedAt: finale})

	require.NoError(t, err)
	assert.True(t, took)
	assert.Equal(t, cprules.SetName("rush"), taken.Name)
	assert.Equal(t, []cprules.SetName{cprules.Default, "rush"}, inForce.names())
	kept, _, err := choices.Choice(t.Context())
	require.NoError(t, err)
	assert.Equal(t, cprules.Choice{Set: "rush", DecidedAt: finale}, kept)
}

func TestTheLatestDecisionWinsAndOneSentAgainChangesNothing(t *testing.T) {
	useCase, inForce := booted(t, keeping(seeded(), rush()), cprules.NewMemoryChoices())
	over := cprules.Choice{Set: cprules.Default, DecidedAt: finale.Add(2 * time.Hour)}

	for _, choice := range []cprules.Choice{
		{Set: "rush", DecidedAt: finale},
		over,
		{Set: "rush", DecidedAt: finale},
		over,
	} {
		_, _, err := useCase.Play(t.Context(), choice)
		require.NoError(t, err)
	}

	assert.Equal(t, []cprules.SetName{cprules.Default, "rush", cprules.Default}, inForce.names())
}

func TestAChoiceOfTheSetInForceTakesNothing(t *testing.T) {
	useCase, inForce := booted(t, keeping(seeded()), cprules.NewMemoryChoices())

	_, took, err := useCase.Play(t.Context(), cprules.Choice{Set: cprules.Default, DecidedAt: finale})

	require.NoError(t, err)
	assert.False(t, took)
	assert.Len(t, inForce.sets, 1)
}

func TestASetTheModuleDoesNotKeepIsRefusedAndTheRulesInForceStay(t *testing.T) {
	choices := cprules.NewMemoryChoices()
	useCase, inForce := booted(t, keeping(seeded()), choices)

	_, took, err := useCase.Play(t.Context(), cprules.Choice{Set: "rush", DecidedAt: finale})

	require.ErrorIs(t, err, cprules.ErrUnknownSet)
	assert.False(t, took)
	assert.Equal(t, []cprules.SetName{cprules.Default}, inForce.names())
	_, kept, err := choices.Choice(t.Context())
	require.NoError(t, err)
	assert.False(t, kept, "a refused choice is not kept")
}

func TestAKeptSetThatCannotPlayIsRefused(t *testing.T) {
	sets := keeping(seeded())
	sets.Keep(named("rush", func(r *planetv1.Rules) { r.Clicks.Burst = proto.Uint32(0) }))
	useCase, inForce := booted(t, sets, cprules.NewMemoryChoices())

	_, _, err := useCase.Play(t.Context(), cprules.Choice{Set: "rush", DecidedAt: finale})

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.Len(t, inForce.sets, 1)
}

func TestAFailedReadKeepsTheRulesInForce(t *testing.T) {
	sets := keeping(seeded(), rush())
	useCase, inForce := booted(t, sets, cprules.NewMemoryChoices())

	down := errors.New("postgres is down")
	sets.FailWith(down)
	_, took, err := useCase.Play(t.Context(), cprules.Choice{Set: "rush", DecidedAt: finale})

	require.ErrorIs(t, err, down)
	assert.False(t, took)
	assert.Equal(t, []cprules.SetName{cprules.Default}, inForce.names())
}

func TestAChoiceThatCannotBeKeptPlaysNothing(t *testing.T) {
	choices := cprules.NewMemoryChoices()
	useCase, inForce := booted(t, keeping(seeded(), rush()), choices)

	choices.FailWith(errors.New("postgres is down"))
	_, _, err := useCase.Play(t.Context(), cprules.Choice{Set: "rush", DecidedAt: finale})

	require.Error(t, err)
	assert.Len(t, inForce.sets, 1)

	choices.FailWith(nil)
	_, took, err := useCase.Play(t.Context(), cprules.Choice{Set: "rush", DecidedAt: finale})
	require.NoError(t, err)
	assert.True(t, took, "the call made again plays it")
}

func TestWithNoDefaultNothingPlays(t *testing.T) {
	inForce := &enforced{}

	_, _, err := cprules.NewPlayUseCase(keeping(), cprules.NewMemoryChoices(), inForce, nil).Execute(t.Context())

	require.ErrorIs(t, err, cprules.ErrUnknownSet, "the boot refuses it")
	assert.Empty(t, inForce.sets)
}

func TestAChangeOfTheSetInForceIsTakenAtTheNextExecute(t *testing.T) {
	sets := keeping(seeded())
	useCase, inForce := booted(t, sets, cprules.NewMemoryChoices())

	_, took, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	assert.False(t, took, "unchanged, it is not taken again")

	sets.Keep(named(cprules.Default, func(r *planetv1.Rules) { r.Clicks.Burst = proto.Uint32(40) }))
	_, took, err = useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.True(t, took)
	require.Len(t, inForce.sets, 2)
	assert.Equal(t, uint32(40), inForce.sets[1].Rules.GetClicks().GetBurst())
}

func TestAKeptSetThatCannotPlayKeepsTheRulesInForceAtTheNextExecute(t *testing.T) {
	sets := keeping(seeded())
	useCase, inForce := booted(t, sets, cprules.NewMemoryChoices())

	sets.Keep(named(cprules.Default, func(r *planetv1.Rules) { r.Clicks.Burst = proto.Uint32(0) }))
	_, took, err := useCase.Execute(t.Context())

	require.ErrorIs(t, err, cprules.ErrInvalidSet)
	assert.False(t, took)
	assert.Len(t, inForce.sets, 1)
}

func TestTheHookHearsEachSetTakenAndNothingElse(t *testing.T) {
	hook := &told{}
	useCase := cprules.NewPlayUseCase(keeping(seeded(), rush()), cprules.NewMemoryChoices(), &enforced{}, hook.hook)

	for range 2 {
		_, _, err := useCase.Execute(t.Context())
		require.NoError(t, err)
	}
	for _, choice := range []cprules.Choice{
		{Set: "rush", DecidedAt: finale},
		{Set: cprules.Default, DecidedAt: finale.Add(2 * time.Hour)},
		{Set: "rush", DecidedAt: finale},
	} {
		_, _, err := useCase.Play(t.Context(), choice)
		require.NoError(t, err)
	}

	assert.Equal(t, []cprules.SetName{cprules.Default, "rush", cprules.Default}, hook.heard(),
		"default at boot, rush, then default; a set unchanged or a choice sent again is not told")
}

func TestTheHookHearsTheSetAgainUntilItHeardIt(t *testing.T) {
	hook := &told{}
	useCase := cprules.NewPlayUseCase(keeping(seeded(), rush()), cprules.NewMemoryChoices(), &enforced{}, hook.hook)
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	down := errors.New("could not tell planet the running bans")
	hook.err = down

	_, took, err := useCase.Play(t.Context(), cprules.Choice{Set: "rush", DecidedAt: finale})
	require.Equal(t, down, err, "the hook's error, which says what failed")
	require.True(t, took, "the set is in force all the same")

	hook.err = nil
	taken, took, err := useCase.Play(t.Context(), cprules.Choice{Set: "rush", DecidedAt: finale})
	require.NoError(t, err, "the call made again")

	assert.False(t, took)
	assert.Equal(t, cprules.SetName("rush"), taken.Name)
	assert.Equal(t, []cprules.SetName{cprules.Default, "rush", "rush"}, hook.heard())
}

func TestAnOperatorsSaveAndTheSchedulesChoiceArePlayedOneAtATime(t *testing.T) {
	hook := &told{}
	rules := inForce()
	useCase := cprules.NewPlayUseCase(keeping(seeded(), rush()), cprules.NewMemoryChoices(), rules, hook.hook)
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	var plays sync.WaitGroup
	for range 10 {
		plays.Go(func() { _, _, _ = useCase.Play(t.Context(), cprules.Choice{Set: "rush", DecidedAt: finale}) })
		plays.Go(func() { _, _, _ = useCase.Execute(t.Context()) })
	}
	plays.Wait()

	assert.Equal(t, []cprules.SetName{cprules.Default, "rush"}, hook.heard(), "default at first, then rush once")
	assert.Equal(t, cprules.SetName("rush"), rules.Name())
}
