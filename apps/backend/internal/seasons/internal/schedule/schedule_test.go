package schedule_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/schedule"
)

var (
	finale     = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	seasonEnds = finale.Add(2 * time.Hour)
	sets       = schedule.PhaseSets{Days: "default", Finale: "rush", Over: "default"}
)

func seasonZero(t *testing.T) schedule.Schedule {
	t.Helper()

	phases := calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: seasonEnds, Finale: 2 * time.Hour}}})
	sched, err := schedule.New(phases.Phases(), sets)
	require.NoError(t, err)
	return sched
}

func id(n byte) schedule.EntryID {
	return schedule.EntryID(uuid.UUID{15: n})
}

func entry(n byte, set schedule.SetName, startsAt, endsAt time.Time, priority schedule.Priority, madeAt time.Time) schedule.Entry {
	return schedule.EntrySnapshot{
		ID: id(n), Set: set, StartsAt: startsAt, EndsAt: endsAt, Priority: priority, MadeAt: madeAt,
	}.Restore()
}

func turn(entry schedule.EntryID, set schedule.SetName) schedule.TurnSnapshot {
	return schedule.TurnSnapshot{Entry: entry, Set: set}
}

func TestThePhasesPlayTheSetOfTheirStage(t *testing.T) {
	sched := seasonZero(t)

	assert.Equal(t, turn(schedule.EntryID{}, "default"), sched.TurnAt(finale.Add(-time.Nanosecond)).TakeSnapshot())
	assert.Equal(t, turn(schedule.EntryID{}, "rush"), sched.TurnAt(finale).TakeSnapshot(), "the finale starts on time")
	assert.Equal(t, turn(schedule.EntryID{}, "rush"), sched.TurnAt(seasonEnds.Add(-time.Nanosecond)).TakeSnapshot())
	assert.Equal(t, turn(schedule.EntryID{}, "default"), sched.TurnAt(seasonEnds).TakeSnapshot(), "over at its end")
	assert.Equal(t, turn(schedule.EntryID{}, "default"), sched.TurnAt(seasonEnds.Add(365*24*time.Hour)).TakeSnapshot())
}

func TestWithNoSeasonAndNoEntryDefaultPlays(t *testing.T) {
	sched, err := schedule.New(nil, sets)
	require.NoError(t, err)

	assert.Equal(t, turn(schedule.EntryID{}, schedule.Default), sched.TurnAt(finale).TakeSnapshot())
	assert.True(t, sched.SinceAt(finale).IsZero())
}

func TestAnEntryAboveThePhasesPlaysWhileItRuns(t *testing.T) {
	sched := seasonZero(t).With(entry(1, "calm", finale.Add(30*time.Minute), finale.Add(time.Hour), 1, finale.Add(-time.Hour)))

	assert.Equal(t, turn(schedule.EntryID{}, "rush"), sched.TurnAt(finale.Add(30*time.Minute-time.Nanosecond)).TakeSnapshot())
	assert.Equal(t, turn(id(1), "calm"), sched.TurnAt(finale.Add(30*time.Minute)).TakeSnapshot())
	assert.Equal(t, turn(schedule.EntryID{}, "rush"), sched.TurnAt(finale.Add(time.Hour)).TakeSnapshot(), "the phase again")
}

func TestTheHighestPriorityPlays(t *testing.T) {
	sched := seasonZero(t).With(
		entry(1, "calm", finale, seasonEnds, 3, finale.Add(-2*time.Hour)),
		entry(2, "frozen", finale, seasonEnds, 2, finale.Add(-time.Hour)),
	)

	assert.Equal(t, turn(id(1), "calm"), sched.TurnAt(finale).TakeSnapshot(), "made first, but above")
}

func TestATieGoesToTheEntryMadeLast(t *testing.T) {
	sched := seasonZero(t).With(
		entry(2, "frozen", finale, seasonEnds, 2, finale.Add(-time.Hour)),
		entry(1, "calm", finale, seasonEnds, 2, finale.Add(-2*time.Hour)),
	)

	assert.Equal(t, turn(id(2), "frozen"), sched.TurnAt(finale).TakeSnapshot())
}

func TestAnEntryThatEndedNoLongerPlays(t *testing.T) {
	sched := seasonZero(t).With(entry(1, "calm", finale.Add(-time.Hour), finale.Add(-time.Minute), 1, finale.Add(-2*time.Hour)))

	assert.Equal(t, turn(schedule.EntryID{}, "rush"), sched.TurnAt(finale).TakeSnapshot())
}

func TestATurnBeganAtTheLastChange(t *testing.T) {
	sched := seasonZero(t).With(
		entry(1, "rush", finale.Add(-time.Hour), finale.Add(time.Hour), 1, finale.Add(-2*time.Hour)),
		entry(2, "calm", finale.Add(10*time.Minute), finale.Add(20*time.Minute), 2, finale.Add(-2*time.Hour)),
	)

	assert.Equal(t, finale.Add(-time.Hour), sched.SinceAt(finale.Add(5*time.Minute)), "the entry, not the phase under it")
	assert.Equal(t, finale.Add(10*time.Minute), sched.SinceAt(finale.Add(15*time.Minute)))
	assert.Equal(t, finale.Add(20*time.Minute), sched.SinceAt(finale.Add(30*time.Minute)), "the entry under it again")
	assert.Equal(t, finale.Add(time.Hour), sched.SinceAt(finale.Add(90*time.Minute)), "the finale's phase")
	assert.Equal(t, seasonEnds, sched.SinceAt(seasonEnds.Add(time.Hour)))
}

func TestTwoPhasesInARowThatPlayOneSetAreOneTurn(t *testing.T) {
	phases := calendar.New(calendar.Config{List: []calendar.Entry{{Number: 0, EndsAt: seasonEnds, Finale: 2 * time.Hour}}})
	sched, err := schedule.New(phases.Phases(), schedule.PhaseSets{Days: "default", Finale: "default", Over: "default"})
	require.NoError(t, err)

	assert.Equal(t, sched.TurnAt(finale.Add(-time.Second)), sched.TurnAt(seasonEnds.Add(time.Second)))
	assert.True(t, sched.SinceAt(seasonEnds.Add(time.Second)).IsZero(), "it plays since before anything")
}
