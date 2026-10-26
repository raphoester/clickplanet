package play_rules_usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

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

var finale = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

type harness struct {
	t       *testing.T
	sets    *rules.MemoryStore
	choices *rules.MemoryChoices
	inForce *rules.InForce
	bans    *told
	useCase *play_rules_usecase.UseCase
}

func newHarness(t *testing.T, kept ...rules.Snapshot) *harness {
	h := &harness{
		t: t, sets: rules.NewMemoryStore(guard.Parts(), kept...), choices: rules.NewMemoryChoices(),
		inForce: rules.NewInForce(), bans: &told{},
	}
	h.useCase = play_rules_usecase.New(h.sets, h.choices, h.inForce, h.bans)
	return h
}

func assertTaken(t *testing.T, want rules.Snapshot, taken rules.Snapshot) {
	t.Helper()
	assert.Equal(t, want.Name, taken.Name)
	assert.True(t, proto.Equal(want.Rules, taken.Rules), "%v", taken.Rules)
}

func (h *harness) execute() (rules.Snapshot, bool) {
	h.t.Helper()
	taken, took, err := h.useCase.Execute(h.t.Context())
	require.NoError(h.t, err)
	return taken, took
}

func (h *harness) play(set rules.SetName, decidedAt time.Time) (rules.Snapshot, bool, error) {
	h.t.Helper()
	return h.useCase.Play(h.t.Context(), rules.Choice{Set: set, DecidedAt: decidedAt}) //nolint:wrapcheck // the test reads the use case's own error.
}

func TestAtFirstDefaultPlaysAndPlanetHearsTheRunningBans(t *testing.T) {
	h := newHarness(t, rules.Seeded())

	taken, took := h.execute()

	require.True(t, took)
	assertTaken(t, rules.Seeded(), taken)
	assert.Equal(t, rules.Default, h.inForce.Name())
	assert.Equal(t, 1, h.bans.Times(), "the bans running at a deploy reach planet's table")
}

func TestABootPlaysTheSetChosenLast(t *testing.T) {
	h := newHarness(t, rules.Seeded(), rush)
	require.NoError(t, h.choices.Save(t.Context(), rules.Choice{Set: "rush", DecidedAt: finale}))

	taken, _ := h.execute()

	assertTaken(t, rush, taken)
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

	h.sets.Keep(rules.Named(rules.Default, func(r *antibotv1.Rules) { r.Bans.Enforce = false }))
	taken, took := h.execute()

	require.True(t, took)
	assert.False(t, taken.Rules.GetBans().GetEnforce())
	assert.False(t, rules.Reader(h.inForce, bans.Part)().Enforce)
	assert.Equal(t, 2, h.bans.Times(), "a set that stops enforcing admits the running bans")
}

func TestAChoicePlaysItsSetAndIsKept(t *testing.T) {
	h := newHarness(t, rules.Seeded(), rush)
	h.execute()

	taken, took, err := h.play("rush", finale)

	require.NoError(t, err)
	require.True(t, took)
	assertTaken(t, rush, taken)
	assert.Equal(t, 50, rules.Reader(h.inForce, cohort.Part)().CertainMembers)
	assert.Equal(t, 2, h.bans.Times())
	kept, _, err := h.choices.Choice(t.Context())
	require.NoError(t, err)
	assert.Equal(t, rules.Choice{Set: "rush", DecidedAt: finale}, kept)
}

func TestTheLatestDecisionWinsAndOneSentAgainChangesNothing(t *testing.T) {
	h := newHarness(t, rules.Seeded(), rush)
	h.execute()

	for _, choice := range []rules.Choice{
		{Set: "rush", DecidedAt: finale},
		{Set: rules.Default, DecidedAt: finale.Add(2 * time.Hour)},
		{Set: "rush", DecidedAt: finale},
	} {
		_, _, err := h.play(choice.Set, choice.DecidedAt)
		require.NoError(t, err)
	}

	assert.Equal(t, rules.Default, h.inForce.Name())
	assert.Equal(t, 3, h.bans.Times(), "default at boot, rush, then default")
}

func TestASetTheAntibotDoesNotKeepIsRefusedAndTheRulesInForceStay(t *testing.T) {
	h := newHarness(t, rules.Seeded())
	h.execute()

	_, took, err := h.play("rush", finale)

	require.ErrorIs(t, err, rules.ErrUnknownSet)
	assert.False(t, took)
	assert.Equal(t, rules.Default, h.inForce.Name())
	_, kept, err := h.choices.Choice(t.Context())
	require.NoError(t, err)
	assert.False(t, kept, "a refused choice is not kept")
	assert.False(t, h.useCase.Plays("rush"), "so a rush can be saved")
}

func TestAFailedReadKeepsTheRulesInForce(t *testing.T) {
	h := newHarness(t, rules.Seeded(), rush)
	h.execute()

	down := errors.New("postgres is down")
	h.sets.FailWith(down)
	_, took, err := h.play("rush", finale)

	require.ErrorIs(t, err, down)
	assert.False(t, took)
	assert.Equal(t, rules.Default, h.inForce.Name())
}

func TestAKeptSetThatCannotPlayKeepsTheRulesInForce(t *testing.T) {
	h := newHarness(t, rules.Seeded())
	h.execute()

	h.sets.Keep(rules.Named(rules.Default, func(r *antibotv1.Rules) { r.Jury.MinSuspects = 0 }))
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
	h := newHarness(t, rules.Seeded(), rush)
	h.execute()
	h.bans.err = errors.New("postgres is down")

	_, took, err := h.play("rush", finale)
	require.Error(t, err)
	require.True(t, took, "the set is in force all the same")

	h.bans.err = nil
	_, took, err = h.play("rush", finale)
	require.NoError(t, err, "the command delivered again")

	assert.False(t, took)
	assert.Equal(t, 3, h.bans.Times())
}

func TestAnOperatorsSaveAndTheSchedulesChoiceArePlayedOneAtATime(t *testing.T) {
	h := newHarness(t, rules.Seeded(), rush)
	h.execute()

	var plays sync.WaitGroup
	for range 10 {
		plays.Go(func() { _, _, _ = h.play("rush", finale) })
		plays.Go(func() { _, _, _ = h.useCase.Execute(t.Context()) })
	}
	plays.Wait()

	assert.Equal(t, 2, h.bans.Times(), "default at first, then rush once")
}

func TestTheSetThatPlaysNowIsTheOneInForce(t *testing.T) {
	h := newHarness(t, rules.Seeded(), rush)
	h.execute()

	assert.True(t, h.useCase.Plays(rules.Default))
	assert.False(t, h.useCase.Plays("rush"))

	_, _, err := h.play("rush", finale)
	require.NoError(t, err)

	assert.False(t, h.useCase.Plays(rules.Default))
	assert.True(t, h.useCase.Plays("rush"))
}
