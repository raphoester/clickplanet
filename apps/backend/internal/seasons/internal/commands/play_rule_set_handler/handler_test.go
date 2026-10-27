package play_rule_set_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	seasonsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/seasons/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/commands/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rules/usecases/play_rules_usecase"
)

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

type opened struct{ startsAt, endsAt time.Time }

type rounds struct{ opened []opened }

func (r *rounds) Execute(_ context.Context, _ uint32, startsAt, endsAt time.Time) error {
	r.opened = append(r.opened, opened{startsAt: startsAt, endsAt: endsAt})
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

func TestTheSetNamedPlaysForItsEntrysWindow(t *testing.T) {
	useCase, choices, opens := booted(t, rules.Battle("rush", 3))

	require.NoError(t, play_rule_set_handler.New(useCase).Handle(t.Context(), &seasonsv1.PlayRuleSet{
		RuleSet: "rush", DecidedAt: timestamppb.New(finale),
		StartsAt: timestamppb.New(finale), EndsAt: timestamppb.New(finale.Add(2 * time.Hour)),
	}))

	assert.True(t, useCase.Plays("rush"))
	assert.Equal(t, []opened{{startsAt: finale, endsAt: finale.Add(2 * time.Hour)}}, opens.opened)
	kept, _, err := choices.Choice(t.Context())
	require.NoError(t, err)
	assert.True(t, finale.Equal(kept.DecidedAt))
}

func TestDefaultPlaysWithNoWindow(t *testing.T) {
	useCase, choices, _ := booted(t)

	require.NoError(t, play_rule_set_handler.New(useCase).Handle(t.Context(), &seasonsv1.PlayRuleSet{
		RuleSet: string(rules.Default), DecidedAt: timestamppb.New(finale),
	}))

	kept, _, err := choices.Choice(t.Context())
	require.NoError(t, err)
	assert.True(t, kept.Window.None())
}

func TestASetSeasonsDoesNotKeepIsRefusedAndDropped(t *testing.T) {
	useCase, _, _ := booted(t)

	require.NoError(t, play_rule_set_handler.New(useCase).Handle(t.Context(), &seasonsv1.PlayRuleSet{
		RuleSet: "rush", DecidedAt: timestamppb.New(finale),
	}))

	assert.True(t, useCase.Plays(rules.Default))
}

func TestAChoiceThatCannotBeKeptIsDeliveredAgain(t *testing.T) {
	useCase, choices, _ := booted(t, rules.Battle("rush", 3))
	down := errors.New("postgres is down")
	choices.FailWith(down)

	err := play_rule_set_handler.New(useCase).Handle(t.Context(), &seasonsv1.PlayRuleSet{
		RuleSet: "rush", DecidedAt: timestamppb.New(finale),
	})

	require.ErrorIs(t, err, down)
}
