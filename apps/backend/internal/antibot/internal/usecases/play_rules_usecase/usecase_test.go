package play_rules_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

var rush = rules.Named("rush", func(s *rules.Snapshot) { s.Cohort.CertainMembers = 50 })

type harness struct {
	t       *testing.T
	sets    *rules.MemoryStore
	inForce *rules.InForce
	bans    *told
	useCase *play_rules_usecase.UseCase
}

func newHarness(t *testing.T, kept ...rules.Snapshot) *harness {
	h := &harness{t: t, sets: rules.NewMemoryStore(kept...), inForce: rules.NewInForce(), bans: &told{}}
	h.useCase = play_rules_usecase.New(h.sets, h.inForce, h.bans)
	return h
}

func (h *harness) save(kept rules.Snapshot) {
	h.t.Helper()
	set, err := kept.Restore()
	require.NoError(h.t, err)
	require.NoError(h.t, h.sets.Save(h.t.Context(), set))
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
	assert.Equal(t, play_rules_usecase.Taken{Set: rules.Seeded(), Asked: rules.Default}, taken)
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

	h.save(rules.Named(rules.Default, func(s *rules.Snapshot) { s.Bans.Enforce = false }))
	taken, took := h.execute()

	require.True(t, took)
	assert.False(t, taken.Set.Bans.Enforce)
	assert.False(t, h.inForce.Bans().Enforce)
	assert.Equal(t, 2, h.bans.Times(), "a set that stops enforcing admits the running bans")
}

func TestTheSetNamedLikePlanetsPlaysOnceTheAntibotKeepsOne(t *testing.T) {
	h := newHarness(t, rules.Seeded())
	h.execute()

	h.useCase.Play("rush")
	taken, took := h.execute()
	require.False(t, took, "no antibot set is named rush yet")
	assert.Equal(t, play_rules_usecase.Taken{Set: rules.Seeded(), Asked: "rush"}, taken)

	h.save(rush)
	taken, took = h.execute()
	require.True(t, took)
	assert.Equal(t, play_rules_usecase.Taken{Set: rush, Asked: "rush"}, taken)
	assert.Equal(t, 50, h.inForce.Cohort().CertainMembers)

	h.useCase.Play(rules.Default)
	taken, took = h.execute()
	require.True(t, took)
	assert.Equal(t, rules.Default, taken.Set.Name)
}

func TestAFailedReadKeepsTheRulesInForce(t *testing.T) {
	h := newHarness(t, rules.Seeded(), rush)
	h.execute()

	down := errors.New("postgres is down")
	h.sets.FailWith(down)
	h.useCase.Play("rush")
	_, took, err := h.useCase.Execute(t.Context())

	require.ErrorIs(t, err, down)
	assert.False(t, took)
	assert.Equal(t, rules.Default, h.inForce.Name())
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

func TestTheRunnerTakesAChangeWithinItsInterval(t *testing.T) {
	h := newHarness(t, rules.Seeded())
	h.execute()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		play_rules_usecase.NewRunner(10*time.Millisecond, h.useCase).Run(ctx)
		close(done)
	}()
	defer func() {
		cancel()
		<-done
	}()

	h.save(rules.Named(rules.Default, func(s *rules.Snapshot) { s.Jury.MinSuspects = 3 }))

	assert.Eventually(t, func() bool { return h.inForce.Jury().MinSuspects == 3 }, time.Second, 5*time.Millisecond)
}
