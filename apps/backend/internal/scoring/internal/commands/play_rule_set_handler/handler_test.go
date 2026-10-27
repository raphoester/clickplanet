package play_rule_set_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	scoringv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/scoring/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/commands/play_rule_set_handler"
	scoringrounds "github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/usecases/play_rules_usecase"
)

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

type opened struct {
	entry            scoringrounds.EntryID
	startsAt, endsAt time.Time
}

type rounds struct{ opened []opened }

func (r *rounds) Execute(_ context.Context, entry scoringrounds.EntryID, _ uint32, startsAt, endsAt time.Time) error {
	r.opened = append(r.opened, opened{entry: entry, startsAt: startsAt, endsAt: endsAt})
	return nil
}

func booted(t *testing.T, kept ...rules.Snapshot) (*play_rules_usecase.UseCase, *rules.MemoryChoices, *rounds) {
	t.Helper()

	sets := rules.NewMemoryStore()
	for _, each := range append([]rules.Snapshot{rules.Seeded()}, kept...) {
		sets.Keep(each)
	}
	choices, opens := rules.NewMemoryChoices(), &rounds{}
	useCase := play_rules_usecase.New(sets, choices, opens)
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	return useCase, choices, opens
}

const entryID = "0192f5a0-7c00-7000-8000-000000000031"

func TestTheSetNamedPlaysForItsEntrysWindow(t *testing.T) {
	useCase, choices, opens := booted(t, rules.Battle("rush", 3))

	require.NoError(t, play_rule_set_handler.New(useCase).Handle(t.Context(), &scoringv1.PlayRuleSet{
		RuleSet: "rush", DecidedAt: timestamppb.New(finale),
		StartsAt: timestamppb.New(finale), EndsAt: timestamppb.New(finale.Add(2 * time.Hour)), EntryId: entryID,
	}))

	assert.True(t, useCase.Plays("rush"))
	entry := scoringrounds.EntryID(uuid.MustParse(entryID))
	assert.Equal(t, []opened{{entry: entry, startsAt: finale, endsAt: finale.Add(2 * time.Hour)}}, opens.opened)
	kept, _, err := choices.Choice(t.Context())
	require.NoError(t, err)
	assert.True(t, finale.Equal(kept.DecidedAt))
}

func TestDefaultPlaysWithNoWindow(t *testing.T) {
	useCase, choices, _ := booted(t)

	require.NoError(t, play_rule_set_handler.New(useCase).Handle(t.Context(), &scoringv1.PlayRuleSet{
		RuleSet: string(rules.Default), DecidedAt: timestamppb.New(finale),
	}))

	kept, _, err := choices.Choice(t.Context())
	require.NoError(t, err)
	assert.True(t, kept.Window.None())
}

func TestAWindowWithNoEntryIsRefusedAndDropped(t *testing.T) {
	useCase, _, opens := booted(t, rules.Battle("rush", 3))

	require.NoError(t, play_rule_set_handler.New(useCase).Handle(t.Context(), &scoringv1.PlayRuleSet{
		RuleSet: "rush", DecidedAt: timestamppb.New(finale),
		StartsAt: timestamppb.New(finale), EndsAt: timestamppb.New(finale.Add(2 * time.Hour)),
	}))

	assert.True(t, useCase.Plays(rules.Default))
	assert.Empty(t, opens.opened)
}

func TestASetScoringDoesNotKeepIsRefusedAndDropped(t *testing.T) {
	useCase, _, _ := booted(t)

	require.NoError(t, play_rule_set_handler.New(useCase).Handle(t.Context(), &scoringv1.PlayRuleSet{
		RuleSet: "rush", DecidedAt: timestamppb.New(finale),
	}))

	assert.True(t, useCase.Plays(rules.Default))
}

func TestAChoiceThatCannotBeKeptIsDeliveredAgain(t *testing.T) {
	useCase, choices, _ := booted(t, rules.Battle("rush", 3))
	down := errors.New("postgres is down")
	choices.FailWith(down)

	err := play_rule_set_handler.New(useCase).Handle(t.Context(), &scoringv1.PlayRuleSet{
		RuleSet: "rush", DecidedAt: timestamppb.New(finale),
	})

	require.ErrorIs(t, err, down)
}
