package play_rules_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/parts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/play_rules_usecase"
)

type enforced struct{ sets []rules.Snapshot }

func (e *enforced) Enforce(set rules.Set) { e.sets = append(e.sets, set.TakeSnapshot()) }

func keeping(t *testing.T, kept ...rules.Snapshot) *rules.MemoryStore {
	t.Helper()

	store := rules.NewMemoryStore()
	for _, set := range kept {
		require.NoError(t, store.Save(t.Context(), set))
	}
	return store
}

func TestAtFirstDefaultPlays(t *testing.T) {
	inForce := &enforced{}

	taken, took, err := play_rules_usecase.New(keeping(t, rules.Seeded()), parts.All(), inForce).Execute(t.Context())

	require.NoError(t, err)
	assert.True(t, took)
	assert.Equal(t, rules.Default, taken.Name)
	require.Len(t, inForce.sets, 1)
	assert.True(t, proto.Equal(rules.Seeded().Rules, inForce.sets[0].Rules))
}

func TestASetUnchangedIsNotTakenAgain(t *testing.T) {
	inForce := &enforced{}
	useCase := play_rules_usecase.New(keeping(t, rules.Seeded()), parts.All(), inForce)
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	_, took, err := useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.False(t, took)
	assert.Len(t, inForce.sets, 1)
}

func TestAChangeOfDefaultIsTakenAtTheNextPlay(t *testing.T) {
	store, inForce := keeping(t, rules.Seeded()), &enforced{}
	useCase := play_rules_usecase.New(store, parts.All(), inForce)
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	require.NoError(t, store.Save(t.Context(), rules.Named(rules.Default, func(r *planetv1.Rules) { r.Clicks.Tempo = 3 })))
	_, took, err := useCase.Execute(t.Context())

	require.NoError(t, err)
	assert.True(t, took)
	assert.InDelta(t, 3.0, inForce.sets[1].Rules.GetClicks().GetTempo(), 1e-9)
}

func TestAFailedReadKeepsTheRulesInForce(t *testing.T) {
	store, inForce := keeping(t, rules.Seeded()), &enforced{}
	useCase := play_rules_usecase.New(store, parts.All(), inForce)
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	store.FailWith(errors.New("postgres is down"))
	_, _, err = useCase.Execute(t.Context())

	require.Error(t, err)
	assert.Len(t, inForce.sets, 1)
	assert.True(t, useCase.Plays(rules.Default))
}

func TestAKeptSetThatCannotPlayKeepsTheRulesInForce(t *testing.T) {
	store, inForce := keeping(t, rules.Seeded()), &enforced{}
	useCase := play_rules_usecase.New(store, parts.All(), inForce)
	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	require.NoError(t, store.Save(t.Context(), rules.Named(rules.Default, func(r *planetv1.Rules) { r.Clicks.Burst = 0 })))
	_, _, err = useCase.Execute(t.Context())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.Len(t, inForce.sets, 1)
}

func TestWithNoDefaultNothingPlays(t *testing.T) {
	inForce := &enforced{}

	_, _, err := play_rules_usecase.New(rules.NewMemoryStore(), parts.All(), inForce).Execute(t.Context())

	require.ErrorIs(t, err, rules.ErrUnknownSet)
	assert.Empty(t, inForce.sets)
}

func TestTheSetThatPlaysNowIsTheOneInForce(t *testing.T) {
	useCase := play_rules_usecase.New(keeping(t, rules.Seeded()), parts.All(), &enforced{})
	assert.False(t, useCase.Plays(rules.Default), "nothing plays before the first play")

	_, _, err := useCase.Execute(t.Context())
	require.NoError(t, err)

	assert.True(t, useCase.Plays(rules.Default))
	assert.False(t, useCase.Plays("rush"))
}
