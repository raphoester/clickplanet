package play_rule_set_handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/commands/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/play_rules_usecase"
)

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

type restated struct{}

func (restated) Restate(context.Context) error { return nil }

func play(set string, decidedAt time.Time) *antibotv1.PlayRuleSet {
	return &antibotv1.PlayRuleSet{RuleSet: set, DecidedAt: timestamppb.New(decidedAt)}
}

func booted(t *testing.T, kept ...rules.Snapshot) (*play_rules_usecase.UseCase, *rules.MemoryChoices, *rules.InForce) {
	t.Helper()

	choices, inForce := rules.NewMemoryChoices(), rules.NewInForce()
	useCase := play_rules_usecase.New(rules.NewMemoryStore(guard.Parts(), append([]rules.Snapshot{rules.Seeded()}, kept...)...),
		choices, inForce, restated{})
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	return useCase, choices, inForce
}

func TestTheSetNamedPlays(t *testing.T) {
	useCase, _, inForce := booted(t, rules.Named("rush", rules.AsIs))

	require.NoError(t, play_rule_set_handler.New(useCase).Handle(t.Context(), play("rush", finale)))

	assert.Equal(t, rules.SetName("rush"), inForce.Name())
}

func TestACommandSentAgainOrOutOfOrderChangesNothing(t *testing.T) {
	useCase, _, inForce := booted(t, rules.Named("rush", rules.AsIs))
	handler := play_rule_set_handler.New(useCase)
	over := play(string(rules.Default), finale.Add(2*time.Hour))

	require.NoError(t, handler.Handle(t.Context(), over))
	require.NoError(t, handler.Handle(t.Context(), play("rush", finale)))
	require.NoError(t, handler.Handle(t.Context(), over))

	assert.Equal(t, rules.Default, inForce.Name())
}

func TestASetTheAntibotDoesNotKeepIsRefusedAndDropped(t *testing.T) {
	useCase, _, inForce := booted(t)

	require.NoError(t, play_rule_set_handler.New(useCase).Handle(t.Context(), play("rush", finale)))

	assert.Equal(t, rules.Default, inForce.Name())
}

func TestAChoiceThatCannotBeKeptIsDeliveredAgain(t *testing.T) {
	useCase, choices, _ := booted(t, rules.Named("rush", rules.AsIs))
	down := errors.New("postgres is down")
	choices.FailWith(down)

	err := play_rule_set_handler.New(useCase).Handle(t.Context(), play("rush", finale))

	require.ErrorIs(t, err, down)
}
