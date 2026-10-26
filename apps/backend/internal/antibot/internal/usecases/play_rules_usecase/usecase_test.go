package play_rules_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	antibotv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/antibot/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/cohort"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/jury"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/play_rules_usecase"
)

type told struct {
	mu    sync.Mutex
	times int
	err   error
}

func (t *told) Restate(context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.times++
	return t.err
}

func (t *told) Times() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.times
}

var rush = rules.Named("rush", func(r *antibotv1.Rules) { r.Cohort.CertainMembers = 50 })

type harness struct {
	t       *testing.T
	sets    *rules.MemoryStore
	inForce *rules.InForce
	bans    *told
	useCase *play_rules_usecase.UseCase
}

func newHarness(t *testing.T, kept ...rules.Snapshot) *harness {
	h := &harness{t: t, sets: rules.NewMemoryStore(kept...), inForce: rules.NewInForce(), bans: &told{}}
	h.useCase = play_rules_usecase.New(h.sets, guard.Parts(), h.inForce, h.bans)
	return h
}

func (h *harness) save(kept rules.Snapshot) {
	h.t.Helper()
	require.NoError(h.t, h.sets.Save(h.t.Context(), kept))
}

func assertTaken(t *testing.T, want rules.Snapshot, asked rules.SetName, taken play_rules_usecase.Taken) {
	t.Helper()
	assert.Equal(t, want.Name, taken.Set.Name)
	assert.True(t, proto.Equal(want.Rules, taken.Set.Rules), "%v", taken.Set.Rules)
	assert.Equal(t, asked, taken.Asked)
}

func (h *harness) execute() (play_rules_usecase.Taken, bool) {
	h.t.Helper()
	taken, took, err := h.useCase.Execute(h.t.Context())
	require.NoError(h.t, err)
	return taken, took
}

func TestAtFirstDefaultPlaysAndPlanetHearsTheRunningBans(t *testing.T) {
	h := newHarness(t, rules.Seeded())

	taken, took := h.execute()

	require.True(t, took)
	assertTaken(t, rules.Seeded(), rules.Default, taken)
	assert.Equal(t, rules.Default, h.inForce.Name())
	assert.Equal(t, 1, h.bans.Times(), "the bans running at a deploy reach planet's table")
}

func TestASetUnchangedIsNotTakenAgain(t *testing.T) {
	h := newHarness(t, rules.Seeded())
	h.execute()

	_, took := h.execute()

	assert.False(t, took)
	assert.Equal(t, 1, h.bans.Times())
}

func TestAChangeOfTheSetInForceIsTakenAtTheNextExecute(t *testing.T) {
	h := newHarness(t, rules.Seeded())
	h.execute()

	h.save(rules.Named(rules.Default, func(r *antibotv1.Rules) { r.Bans.Enforce = false }))
	taken, took := h.execute()

	require.True(t, took)
	assert.False(t, taken.Set.Rules.GetBans().GetEnforce())
	assert.False(t, rules.Reader(h.inForce, bans.Part)().Enforce)
	assert.Equal(t, 2, h.bans.Times(), "a set that stops enforcing admits the running bans")
}

func TestTheSetNamedLikePlanetsPlaysOnceTheAntibotKeepsOne(t *testing.T) {
	h := newHarness(t, rules.Seeded())
	h.execute()

	taken, took, err := h.useCase.Play(t.Context(), "rush")
	require.NoError(t, err)
	require.False(t, took, "no antibot set is named rush yet")
	assertTaken(t, rules.Seeded(), "rush", taken)

	h.save(rush)
	taken, took = h.execute()
	require.True(t, took)
	assertTaken(t, rush, "rush", taken)
	assert.Equal(t, 50, rules.Reader(h.inForce, cohort.Part)().CertainMembers)

	taken, took, err = h.useCase.Play(t.Context(), rules.Default)
	require.NoError(t, err)
	require.True(t, took)
	assert.Equal(t, rules.Default, taken.Set.Name)
}

func TestAFailedReadKeepsTheRulesInForce(t *testing.T) {
	h := newHarness(t, rules.Seeded(), rush)
	h.execute()

	down := errors.New("postgres is down")
	h.sets.FailWith(down)
	_, took, err := h.useCase.Play(t.Context(), "rush")

	require.ErrorIs(t, err, down)
	assert.False(t, took)
	assert.Equal(t, rules.Default, h.inForce.Name())
}

func TestAKeptSetThatCannotPlayKeepsTheRulesInForce(t *testing.T) {
	h := newHarness(t, rules.Seeded())
	h.execute()

	h.save(rules.Named(rules.Default, func(r *antibotv1.Rules) { r.Jury.MinSuspects = 0 }))
	_, took, err := h.useCase.Execute(t.Context())

	require.ErrorIs(t, err, rules.ErrInvalidSet)
	assert.False(t, took)
	assert.Equal(t, 2, rules.Reader(h.inForce, jury.Part)().MinSuspects)
}

func TestWithNoDefaultNothingPlays(t *testing.T) {
	h := newHarness(t)

	_, _, err := h.useCase.Execute(t.Context())

	require.ErrorIs(t, err, rules.ErrUnknownSet, "the boot refuses it")
}

func TestPlanetHearsTheBansAgainUntilItHasHeardThem(t *testing.T) {
	h := newHarness(t, rules.Seeded())
	h.bans.err = errors.New("postgres is down")

	_, took, err := h.useCase.Execute(t.Context())
	require.Error(t, err)
	require.True(t, took, "the set is in force all the same")

	h.bans.err = nil
	_, took = h.execute()

	assert.False(t, took)
	assert.Equal(t, 2, h.bans.Times())
}

func TestAnOperatorsSaveAndPlanetsNameArePlayedOneAtATime(t *testing.T) {
	h := newHarness(t, rules.Seeded(), rush)
	h.execute()

	var plays sync.WaitGroup
	for range 10 {
		plays.Go(func() { _, _, _ = h.useCase.Play(t.Context(), "rush") })
		plays.Go(func() { _, _, _ = h.useCase.Execute(t.Context()) })
	}
	plays.Wait()

	assert.Equal(t, rules.SetName("rush"), h.inForce.Name())
	assert.Equal(t, 2, h.bans.Times(), "default at first, then rush once")
}

func TestTheSetsThatPlayNowAreTheOneInForceAndTheOnePlanetNames(t *testing.T) {
	h := newHarness(t, rules.Seeded())
	h.execute()

	assert.True(t, h.useCase.Plays(rules.Default))
	assert.False(t, h.useCase.Plays("rush"))

	_, _, err := h.useCase.Play(t.Context(), "rush")
	require.NoError(t, err)

	assert.True(t, h.useCase.Plays(rules.Default), "in force while the antibot keeps no rush")
	assert.True(t, h.useCase.Plays("rush"), "a rush saved now would play at once")
	assert.False(t, h.useCase.Plays("calm"))
}
