package play_schedule_usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule/inmemory_entry_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule/usecases/play_schedule_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	finale     = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	seasonEnds = finale.Add(2 * time.Hour)
)

type command struct {
	set       schedule.SetName
	decidedAt time.Time
}

type commanded struct {
	sent []command
	err  error
}

func (c *commanded) Play(_ context.Context, turn schedule.Turn, decidedAt time.Time) error {
	if c.err != nil {
		return c.err
	}
	c.sent = append(c.sent, command{set: turn.TakeSnapshot().Set, decidedAt: decidedAt})
	return nil
}

func (c *commanded) sets() []schedule.SetName {
	sets := make([]schedule.SetName, 0, len(c.sent))
	for _, each := range c.sent {
		sets = append(sets, each.set)
	}
	return sets
}

type harness struct {
	t        *testing.T
	clock    *cptime.FixedClock
	entries  *inmemory_entry_store.Store
	commands *commanded
	useCase  *play_schedule_usecase.UseCase
}

func newHarness(t *testing.T, at time.Time) *harness {
	t.Helper()

	phases, err := schedule.New(calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: seasonEnds, Finale: 2 * time.Hour},
	}}).Phases(), schedule.PhaseSets{Days: "default", Finale: "rush", Over: "default"})
	require.NoError(t, err)

	h := &harness{t: t, clock: cptime.NewFixedClock(at), entries: inmemory_entry_store.New(), commands: &commanded{}}
	h.useCase = play_schedule_usecase.New(phases, h.entries, h.commands, h.clock)
	return h
}

func (h *harness) execute() bool {
	h.t.Helper()
	_, told, err := h.useCase.Execute(h.t.Context())
	require.NoError(h.t, err)
	return told
}

func (h *harness) schedule(n byte, set schedule.SetName, startsAt, endsAt time.Time, priority schedule.Priority) {
	h.t.Helper()
	require.NoError(h.t, h.entries.Save(h.t.Context(), schedule.EntrySnapshot{
		ID: schedule.EntryID(uuid.UUID{15: n}), Set: set, StartsAt: startsAt, EndsAt: endsAt, Priority: priority,
		MadeAt: finale.Add(-24 * time.Hour),
	}.Restore()))
}

func TestTheFirstExecuteCommandsTheTurnAtNow(t *testing.T) {
	h := newHarness(t, finale.Add(-time.Hour))

	assert.True(t, h.execute(), "planet and the antibot may have kept a stale set")

	assert.Equal(t, []command{{set: "default", decidedAt: finale.Add(-time.Hour)}}, h.commands.sent)
}

func TestNothingIsSentWhileTheTurnHolds(t *testing.T) {
	h := newHarness(t, finale.Add(-time.Hour))
	h.execute()

	for range 3 {
		h.clock.Advance(time.Second)
		assert.False(t, h.execute())
	}

	assert.Len(t, h.commands.sent, 1)
}

func TestEachChangeOfTurnIsSentOnceAtItsTime(t *testing.T) {
	h := newHarness(t, finale.Add(-time.Second))
	h.execute()

	h.clock.Advance(time.Second)
	assert.True(t, h.execute(), "the finale starts")
	h.clock.Advance(time.Second)
	assert.False(t, h.execute())
	h.clock.Advance(2 * time.Hour)
	assert.True(t, h.execute(), "the season is over")

	assert.Equal(t, []command{
		{set: "default", decidedAt: finale.Add(-time.Second)},
		{set: "rush", decidedAt: finale},
		{set: "default", decidedAt: finale.Add(2*time.Hour + time.Second)},
	}, h.commands.sent)
}

func TestAnEntryAboveThePhaseIsSentWhenItStartsAndWhenItEnds(t *testing.T) {
	h := newHarness(t, finale.Add(time.Minute))
	h.execute()
	h.schedule(1, "calm", finale.Add(2*time.Minute), finale.Add(3*time.Minute), 1)

	h.clock.Advance(time.Minute)
	h.execute()
	h.clock.Advance(time.Minute)
	h.execute()

	assert.Equal(t, []schedule.SetName{"rush", "calm", "rush"}, h.commands.sets())
}

func TestAnEntryOfTheSetThatPlaysIsSentAgain(t *testing.T) {
	h := newHarness(t, finale.Add(time.Minute))
	h.execute()
	h.schedule(1, "rush", finale.Add(2*time.Minute), seasonEnds, 1)

	h.clock.Advance(time.Minute)

	assert.True(t, h.execute(), "a set a module refused before is sent again with the entry that fixes it")
	assert.Equal(t, []schedule.SetName{"rush", "rush"}, h.commands.sets())
}

func TestACommandRedisDidNotTakeIsSentAtTheNextExecute(t *testing.T) {
	h := newHarness(t, finale)
	down := errors.New("redis is down")
	h.commands.err = down

	_, told, err := h.useCase.Execute(t.Context())
	require.ErrorIs(t, err, down)
	require.False(t, told)

	h.commands.err = nil
	h.clock.Advance(time.Second)
	assert.True(t, h.execute())
	assert.Equal(t, []command{{set: "rush", decidedAt: finale.Add(time.Second)}}, h.commands.sent)
}

func TestAScheduleThatCannotBeReadSendsNothing(t *testing.T) {
	h := newHarness(t, finale)
	down := errors.New("postgres is down")
	h.entries.FailWith(down)

	_, told, err := h.useCase.Execute(t.Context())

	require.ErrorIs(t, err, down)
	assert.False(t, told)
	assert.Empty(t, h.commands.sent)
}
