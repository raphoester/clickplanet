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
		ID: id(n), Set: set, Kind: schedule.Battle, Description: schedule.Description(set) + " rules", StartsAt: startsAt,
		EndsAt: endsAt, Priority: priority, MadeAt: madeAt,
	}.Restore()
}

func turn(entry schedule.EntryID, set schedule.SetName) schedule.TurnSnapshot {
	return schedule.TurnSnapshot{Entry: entry, Set: set, Kind: schedule.Battle, Description: schedule.Description(set) + " rules"}
}

// played is a turn's snapshot with no window: TestATurnCarriesItsEntrysWindow pins the window.
func played(turn schedule.Turn) schedule.TurnSnapshot {
	snapshot := turn.TakeSnapshot()
	snapshot.StartsAt, snapshot.EndsAt = time.Time{}, time.Time{}
	return snapshot
}

var (
	none        = schedule.EntryID{}
	defaultTurn = schedule.TurnSnapshot{Entry: none, Set: schedule.Default, Kind: schedule.Plain}
)

func TestWithNoEntryDefaultPlaysAsAPlainTurn(t *testing.T) {
	sched := schedule.New()

	assert.Equal(t, defaultTurn, played(sched.TurnAt(finale)))
	assert.True(t, sched.SinceAt(finale).IsZero())
}

func TestAnEntryPlaysFromItsStartToJustBeforeItsEnd(t *testing.T) {
	sched := schedule.New(entry(1, "rush", finale, battleEnds, 1, madeAt))

	assert.Equal(t, defaultTurn, played(sched.TurnAt(finale.Add(-time.Nanosecond))))
	assert.Equal(t, turn(id(1), "rush"), played(sched.TurnAt(finale)), "the battle starts on time")
	assert.Equal(t, turn(id(1), "rush"), played(sched.TurnAt(battleEnds.Add(-time.Nanosecond))))
	assert.Equal(t, defaultTurn, played(sched.TurnAt(battleEnds)), "over at its end")
}

func TestTheTurnIsOfTheEntrysKindWithItsWords(t *testing.T) {
	plain := entry(1, "calm", finale, battleEnds, 1, madeAt).TakeSnapshot()
	plain.Kind, plain.Description = schedule.Plain, "A quieter antibot."
	sched := schedule.New(plain.Restore())

	assert.Equal(t, schedule.TurnSnapshot{Entry: id(1), Set: "calm", Kind: schedule.Plain, Description: "A quieter antibot."},
		played(sched.TurnAt(finale)))
}

func TestATurnCarriesItsEntrysWindowInUTC(t *testing.T) {
	paris := time.FixedZone("CET", 3600)
	sched := schedule.New(entry(1, "rush", finale.In(paris), battleEnds.In(paris), 1, madeAt))

	snapshot := sched.TurnAt(finale).TakeSnapshot()

	assert.Equal(t, finale, snapshot.StartsAt)
	assert.Equal(t, battleEnds, snapshot.EndsAt)
	assert.Equal(t, sched.TurnAt(finale), schedule.New(entry(1, "rush", finale, battleEnds, 1, madeAt)).TurnAt(finale),
		"two reads of one entry are one turn")
	assert.True(t, schedule.New().TurnAt(finale).TakeSnapshot().EndsAt.IsZero(), "default has no window")
}

func TestTheNextBattleIsTheBattleEntryThatStartsFirstAfterATime(t *testing.T) {
	plain := entry(3, "calm", finale.Add(-2*time.Hour), finale.Add(-time.Hour), 1, madeAt).TakeSnapshot()
	plain.Kind = schedule.Plain
	sched := schedule.New(
		entry(1, "rush", finale, battleEnds, 1, madeAt),
		entry(2, "rush", finale.Add(24*time.Hour), battleEnds.Add(24*time.Hour), 1, madeAt),
		plain.Restore(),
	)

	next, found := sched.NextOfKind(schedule.Battle, finale.Add(-3*time.Hour))
	assert.True(t, found)
	assert.Equal(t, id(1), next.ID, "a plain entry is no battle")

	next, found = sched.NextOfKind(schedule.Battle, finale)
	assert.True(t, found)
	assert.Equal(t, id(2), next.ID, "one that started is not to come")

	_, found = sched.NextOfKind(schedule.Battle, finale.Add(24*time.Hour))
	assert.False(t, found)
}

func TestTheHighestPriorityPlays(t *testing.T) {
	sched := schedule.New(
		entry(1, "calm", finale, battleEnds, 3, madeAt.Add(-time.Hour)),
		entry(2, "frozen", finale, battleEnds, 2, madeAt),
	)

	assert.Equal(t, turn(id(1), "calm"), played(sched.TurnAt(finale)), "made first, but above")
}

func TestAnEntryAboveAnotherPlaysWhileItRuns(t *testing.T) {
	sched := schedule.New(
		entry(1, "rush", finale, battleEnds, 1, madeAt),
		entry(2, "calm", finale.Add(30*time.Minute), finale.Add(time.Hour), 2, madeAt),
	)

	assert.Equal(t, turn(id(1), "rush"), played(sched.TurnAt(finale.Add(30*time.Minute-time.Nanosecond))))
	assert.Equal(t, turn(id(2), "calm"), played(sched.TurnAt(finale.Add(30*time.Minute))))
	assert.Equal(t, turn(id(1), "rush"), played(sched.TurnAt(finale.Add(time.Hour))), "the one under it again")
}

func TestATieGoesToTheEntryMadeLast(t *testing.T) {
	sched := schedule.New(
		entry(2, "frozen", finale, battleEnds, 2, madeAt),
		entry(1, "calm", finale, battleEnds, 2, madeAt.Add(-time.Hour)),
	)

	assert.Equal(t, turn(id(2), "frozen"), played(sched.TurnAt(finale)))
}

func TestATieMadeAtOnceGoesToTheGreaterID(t *testing.T) {
	sched := schedule.New(
		entry(1, "calm", finale, battleEnds, 2, madeAt),
		entry(2, "frozen", finale, battleEnds, 2, madeAt),
	)

	assert.Equal(t, turn(id(2), "frozen"), played(sched.TurnAt(finale)))
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
