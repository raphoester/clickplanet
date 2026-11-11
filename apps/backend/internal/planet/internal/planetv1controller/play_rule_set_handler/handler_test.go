package play_rule_set_handler_test

import (
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/planetv1controller/play_rule_set_handler"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/play_rules_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

func booted(t *testing.T, kept ...rules.Snapshot) (play_rule_set_handler.PlayRuleSetHandler, *rules.MemoryChoices, *rules.InForce) {
	t.Helper()

	sets := rules.NewMemoryStore(parts.All())
	for _, each := range append([]rules.Snapshot{rules.Seeded()}, kept...) {
		sets.Keep(each)
	}
	choices, inForce := rules.NewMemoryChoices(), rules.NewInForce(cptime.NewFixedClock(finale))
	useCase := play_rules_usecase.New(sets, choices, inForce)
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)
	return play_rule_set_handler.New(useCase), choices, inForce
}

func play(t *testing.T, handler play_rule_set_handler.PlayRuleSetHandler, set string, decidedAt time.Time) error {
	t.Helper()

	_, err := handler.PlayRuleSet(t.Context(), connect.NewRequest(&planetv1.PlayRuleSetRequest{
		RuleSet: set, DecidedAt: timestamppb.New(decidedAt),
	}))
	return err //nolint:wrapcheck // the handler's answer, as the caller reads it.
}

func TestTheSetNamedPlays(t *testing.T) {
	handler, choices, inForce := booted(t, rules.Named("rush", rules.AsIs))

	require.NoError(t, play(t, handler, "rush", finale))

	assert.Equal(t, rules.SetName("rush"), inForce.Name())
	kept, _, err := choices.Choice(t.Context())
	require.NoError(t, err)
	assert.True(t, finale.Equal(kept.DecidedAt))
}

func TestACallMadeAgainOrOutOfOrderChangesNothing(t *testing.T) {
	handler, _, inForce := booted(t, rules.Named("rush", rules.AsIs))
	later := finale.Add(2 * time.Hour)

	require.NoError(t, play(t, handler, string(rules.Default), later))
	require.NoError(t, play(t, handler, "rush", finale))
	require.NoError(t, play(t, handler, string(rules.Default), later))

	assert.Equal(t, rules.Default, inForce.Name())
}

func TestASetPlanetDoesNotKeepIsAFailedPrecondition(t *testing.T) {
	handler, _, inForce := booted(t)

	err := play(t, handler, "rush", finale)

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.ErrorIs(t, err, rules.ErrUnknownSet)
	assert.Equal(t, rules.Default, inForce.Name())
}

func TestASetThatCannotPlayIsAFailedPrecondition(t *testing.T) {
	handler, _, inForce := booted(t, rules.Named("rush", func(r *planetv1.Rules) { r.Clicks.Burst = proto.Uint32(0) }))

	err := play(t, handler, "rush", finale)

	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Equal(t, rules.Default, inForce.Name())
}

func TestAChoiceThatCannotBeKeptIsPassedOnForTheErrorNet(t *testing.T) {
	handler, choices, _ := booted(t, rules.Named("rush", rules.AsIs))
	down := errors.New("postgres is down")
	choices.FailWith(down)

	require.ErrorIs(t, play(t, handler, "rush", finale), down)
}
