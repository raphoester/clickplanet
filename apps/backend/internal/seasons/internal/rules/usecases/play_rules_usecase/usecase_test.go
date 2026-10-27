package play_rules_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules/usecases/play_rules_usecase"
)

var (
	finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	battle = rules.Window{StartsAt: finale, EndsAt: finale.Add(2 * time.Hour)}
	rush   = rules.Battle("rush", 3)
)

type opened struct {
	times            uint32
	startsAt, endsAt time.Time
}

type rounds struct {
	opened  []opened
	failing error
}

func (r *rounds) Execute(_ context.Context, times uint32, startsAt, endsAt time.Time) error {
	if r.failing != nil {
		return r.failing
	}
	r.opened = append(r.opened, opened{times: times, startsAt: startsAt, endsAt: endsAt})
	return nil
}

func keeping(t *testing.T, kept ...rules.Snapshot) *rules.MemoryStore {
	t.Helper()

	store := rules.NewMemoryStore()
	for _, each := range kept {
		set, err := each.Restore()
		require.NoError(t, err)
		require.NoError(t, store.Save(t.Context(), set))
	}
	return store
}

func booted(t *testing.T, sets *rules.MemoryStore, choices *rules.MemoryChoices, opens *rounds) *play_rules_usecase.UseCase {
	t.Helper()

	useCase := play_rules_usecase.New(sets, choices, opens)
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	return useCase
}

func TestAtFirstDefaultPlaysAndOpensNothing(t *testing.T) {
	opens := &rounds{}

	played, took, err := play_rules_usecase.New(keeping(t, rules.Seeded()), rules.NewMemoryChoices(), opens).
		Execute(t.Context())

	require.NoError(t, err)
	assert.True(t, took)
	assert.Equal(t, rules.Default, played.Set.Name)
	assert.True(t, played.Window.None())
	assert.Empty(t, opens.opened)
}

func TestABootPlaysTheSetChosenLastAndOpensItsRoundAgain(t *testing.T) {
	choices := rules.NewMemoryChoices()
	require.NoError(t, choices.Save(t.Context(), rules.Choice{Set: "rush", Window: battle, DecidedAt: finale}))
	opens := &rounds{}

	useCase := booted(t, keeping(t, rules.Seeded(), rush), choices, opens)

	assert.True(t, useCase.Plays("rush"))
	assert.Equal(t, []opened{{times: 3, startsAt: battle.StartsAt, endsAt: battle.EndsAt}}, opens.opened)
}

func TestAChoicePlaysItsSetOpensItsRoundAndIsKept(t *testing.T) {
	choices := rules.NewMemoryChoices()
	opens := &rounds{}
	useCase := booted(t, keeping(t, rules.Seeded(), rush), choices, opens)

	played, took, err := useCase.Play(t.Context(), rules.Choice{Set: "rush", Window: battle, DecidedAt: finale})

	require.NoError(t, err)
	assert.True(t, took)
	assert.Equal(t, rules.SetName("rush"), played.Set.Name)
	assert.Equal(t, battle, played.Window)
	assert.Equal(t, rules.SetName("rush"), useCase.Name())
	assert.Equal(t, []opened{{times: 3, startsAt: battle.StartsAt, endsAt: battle.EndsAt}}, opens.opened)
	kept, _, err := choices.Choice(t.Context())
	require.NoError(t, err)
	assert.Equal(t, rules.SetName("rush"), kept.Set)
}

func TestTheSameSetForAnotherEntryOpensAnotherRound(t *testing.T) {
	opens := &rounds{}
	useCase := booted(t, keeping(t, rules.Seeded(), rush), rules.NewMemoryChoices(), opens)
	later := rules.Window{StartsAt: battle.EndsAt, EndsAt: battle.EndsAt.Add(time.Hour)}

	_, _, err := useCase.Play(t.Context(), rules.Choice{Set: "rush", Window: battle, DecidedAt: finale})
	require.NoError(t, err)
	_, _, err = useCase.Play(t.Context(), rules.Choice{Set: "rush", Window: later, DecidedAt: battle.EndsAt})
	require.NoError(t, err)

	require.Len(t, opens.opened, 2)
	assert.Equal(t, later.EndsAt, opens.opened[1].endsAt)
}

func TestASetWithNoRoundOfItsOwnOpensNone(t *testing.T) {
	opens := &rounds{}
	useCase := booted(t, keeping(t, rules.Seeded()), rules.NewMemoryChoices(), opens)

	_, took, err := useCase.Play(t.Context(), rules.Choice{Set: rules.Default, Window: battle, DecidedAt: finale})

	require.NoError(t, err)
	assert.True(t, took)
	assert.Empty(t, opens.opened)
}

func TestADecisionSentAgainOrOlderChangesNothing(t *testing.T) {
	opens := &rounds{}
	useCase := booted(t, keeping(t, rules.Seeded(), rush), rules.NewMemoryChoices(), opens)
	_, _, err := useCase.Play(t.Context(), rules.Choice{Set: "rush", Window: battle, DecidedAt: finale})
	require.NoError(t, err)

	for _, at := range []time.Time{finale, finale.Add(-time.Second)} {
		played, took, err := useCase.Play(t.Context(), rules.Choice{Set: rules.Default, DecidedAt: at})

		require.NoError(t, err)
		assert.False(t, took)
		assert.Equal(t, rules.SetName("rush"), played.Set.Name)
	}
	assert.Len(t, opens.opened, 1)
}

func TestASetSeasonsKeepsNoneOfIsRefusedAndNotKept(t *testing.T) {
	choices := rules.NewMemoryChoices()
	useCase := booted(t, keeping(t, rules.Seeded()), choices, &rounds{})

	played, took, err := useCase.Play(t.Context(), rules.Choice{Set: "rush", Window: battle, DecidedAt: finale})

	require.ErrorIs(t, err, rules.ErrUnknownSet)
	assert.False(t, took)
	assert.Equal(t, rules.Default, played.Set.Name)
	_, kept, err := choices.Choice(t.Context())
	require.NoError(t, err)
	assert.False(t, kept)
}

func TestARoundThatFailedToOpenKeepsNothingSoTheCommandSentAgainOpensIt(t *testing.T) {
	choices := rules.NewMemoryChoices()
	opens := &rounds{}
	useCase := booted(t, keeping(t, rules.Seeded(), rush), choices, opens)
	opens.failing = errors.New("postgres is down")
	choice := rules.Choice{Set: "rush", Window: battle, DecidedAt: finale}

	_, _, err := useCase.Play(t.Context(), choice)
	require.Error(t, err)
	assert.True(t, useCase.Plays(rules.Default))
	_, kept, _ := choices.Choice(t.Context())
	assert.False(t, kept)

	opens.failing = nil
	_, took, err := useCase.Play(t.Context(), choice)
	require.NoError(t, err)
	assert.True(t, took)
	assert.Len(t, opens.opened, 1)
}

func TestABootWithNoDefaultIsRefused(t *testing.T) {
	_, _, err := play_rules_usecase.New(keeping(t), rules.NewMemoryChoices(), &rounds{}).Execute(t.Context())

	require.ErrorIs(t, err, rules.ErrUnknownSet)
}
