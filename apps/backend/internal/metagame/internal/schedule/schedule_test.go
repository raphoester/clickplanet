package schedule_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
)

var (
	finale     = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)
	battleEnds = finale.Add(2 * time.Hour)
	madeAt     = finale.Add(-24 * time.Hour)
)

func id(n byte) schedule.EntryID {
	return schedule.EntryID(uuid.UUID{15: n})
}

func entry(n byte, set schedule.SetName, startsAt, endsAt time.Time, priority schedule.Priority, madeAt time.Time) schedule.Entry {
	return schedule.EntrySnapshot{
		ID: id(n), Set: set, Kind: schedule.Battle, StartsAt: startsAt, EndsAt: endsAt, Priority: priority, MadeAt: madeAt,
	}.Restore()
}

func turn(entry schedule.EntryID, set schedule.SetName) schedule.TurnSnapshot {
	return schedule.TurnSnapshot{Entry: entry, Set: set, Kind: schedule.Battle}
}

var (
	none        = schedule.EntryID{}
	defaultTurn = schedule.TurnSnapshot{Entry: none, Set: schedule.Default, Kind: schedule.Plain}
)

func TestWithNoEntryDefaultPlaysAsAPlainTurn(t *testing.T) {
	sched := schedule.New()

	assert.Equal(t, defaultTurn, sched.TurnAt(finale).TakeSnapshot())
	assert.True(t, sched.SinceAt(finale).IsZero())
}

func TestAnEntryPlaysFromItsStartToJustBeforeItsEnd(t *testing.T) {
	sched := schedule.New(entry(1, "rush", finale, battleEnds, 1, madeAt))

	assert.Equal(t, defaultTurn, sched.TurnAt(finale.Add(-time.Nanosecond)).TakeSnapshot())
	assert.Equal(t, turn(id(1), "rush"), sched.TurnAt(finale).TakeSnapshot(), "the battle starts on time")
	assert.Equal(t, turn(id(1), "rush"), sched.TurnAt(battleEnds.Add(-time.Nanosecond)).TakeSnapshot())
	assert.Equal(t, defaultTurn, sched.TurnAt(battleEnds).TakeSnapshot(), "over at its end")
}

func TestTheTurnIsOfTheEntrysKind(t *testing.T) {
	plain := entry(1, "calm", finale, battleEnds, 1, madeAt).TakeSnapshot()
	plain.Kind = schedule.Plain
	sched := schedule.New(plain.Restore())

	assert.Equal(t, schedule.TurnSnapshot{Entry: id(1), Set: "calm", Kind: schedule.Plain}, sched.TurnAt(finale).TakeSnapshot())
}

func TestTheHighestPriorityPlays(t *testing.T) {
	sched := schedule.New(
		entry(1, "calm", finale, battleEnds, 3, madeAt.Add(-time.Hour)),
		entry(2, "frozen", finale, battleEnds, 2, madeAt),
	)

	assert.Equal(t, turn(id(1), "calm"), sched.TurnAt(finale).TakeSnapshot(), "made first, but above")
}

func TestAnEntryAboveAnotherPlaysWhileItRuns(t *testing.T) {
	sched := schedule.New(
		entry(1, "rush", finale, battleEnds, 1, madeAt),
		entry(2, "calm", finale.Add(30*time.Minute), finale.Add(time.Hour), 2, madeAt),
	)

	assert.Equal(t, turn(id(1), "rush"), sched.TurnAt(finale.Add(30*time.Minute-time.Nanosecond)).TakeSnapshot())
	assert.Equal(t, turn(id(2), "calm"), sched.TurnAt(finale.Add(30*time.Minute)).TakeSnapshot())
	assert.Equal(t, turn(id(1), "rush"), sched.TurnAt(finale.Add(time.Hour)).TakeSnapshot(), "the one under it again")
}

func TestATieGoesToTheEntryMadeLast(t *testing.T) {
	sched := schedule.New(
		entry(2, "frozen", finale, battleEnds, 2, madeAt),
		entry(1, "calm", finale, battleEnds, 2, madeAt.Add(-time.Hour)),
	)

	assert.Equal(t, turn(id(2), "frozen"), sched.TurnAt(finale).TakeSnapshot())
}

func TestATieMadeAtOnceGoesToTheGreaterID(t *testing.T) {
	sched := schedule.New(
		entry(1, "calm", finale, battleEnds, 2, madeAt),
		entry(2, "frozen", finale, battleEnds, 2, madeAt),
	)

	assert.Equal(t, turn(id(2), "frozen"), sched.TurnAt(finale).TakeSnapshot())
}

func TestATurnBeganAtTheLastChange(t *testing.T) {
	sched := schedule.New(
		entry(1, "rush", finale.Add(-time.Hour), finale.Add(time.Hour), 1, madeAt),
		entry(2, "calm", finale.Add(10*time.Minute), finale.Add(20*time.Minute), 2, madeAt),
	)

	assert.Equal(t, finale.Add(-time.Hour), sched.SinceAt(finale.Add(5*time.Minute)))
	assert.Equal(t, finale.Add(10*time.Minute), sched.SinceAt(finale.Add(15*time.Minute)))
	assert.Equal(t, finale.Add(20*time.Minute), sched.SinceAt(finale.Add(30*time.Minute)), "the entry under it again")
	assert.Equal(t, finale.Add(time.Hour), sched.SinceAt(finale.Add(90*time.Minute)), "default again")
}

func TestTwoEntriesOfOneSetInARowAreTwoTurns(t *testing.T) {
	sched := schedule.New(
		entry(1, "rush", finale, finale.Add(time.Hour), 1, madeAt),
		entry(2, "rush", finale.Add(time.Hour), battleEnds, 1, madeAt),
	)

	assert.NotEqual(t, sched.TurnAt(finale), sched.TurnAt(finale.Add(time.Hour)), "the second is commanded again")
	assert.Equal(t, finale.Add(time.Hour), sched.SinceAt(finale.Add(90*time.Minute)))
}
