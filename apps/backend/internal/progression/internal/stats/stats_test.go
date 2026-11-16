package stats_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
)

var beforeMidnight = time.Date(2026, 9, 17, 23, 59, 59, 0, time.UTC)

func taken(at ...time.Time) stats.Stats {
	var counted stats.Stats
	for position, take := range at {
		counted = counted.WithTake(uint64(position), 0, take)
	}
	return counted
}

func TestTheFirstTakeStartsAStreakOfOne(t *testing.T) {
	counted := taken(beforeMidnight)

	assert.Equal(t, stats.StatsSnapshot{
		TilesTaken: 1, Streak: stats.StreakOf(1, stats.DayOf(beforeMidnight)), StreakBest: 1, DaysPlayed: 1,
		Counted: stats.Counted{TakesBelow: stats.TakeMark{Position: 0, Index: 1}},
	}.Restore(), counted)
}

func TestATakeOnTheSameDayCountsATileAndNotADay(t *testing.T) {
	counted := taken(beforeMidnight.Add(-23*time.Hour), beforeMidnight)

	assert.Equal(t, uint64(2), counted.TilesTaken())
	assert.Equal(t, uint32(1), counted.Streak().Days())
}

func TestATakeJustAfterUTCMidnightExtendsTheStreak(t *testing.T) {
	counted := taken(beforeMidnight, beforeMidnight.Add(2*time.Second))

	assert.Equal(t, uint32(2), counted.Streak().Days())
	assert.Equal(t, uint32(2), counted.StreakBest())
	assert.Equal(t, "2026-09-18", counted.Streak().LastDay().String())
}

func TestTheDayIsUTCWhateverTheZoneOfTheTake(t *testing.T) {
	paris := time.FixedZone("CEST", 2*60*60)
	counted := taken(beforeMidnight, time.Date(2026, 9, 18, 0, 30, 0, 0, paris))

	assert.Equal(t, uint32(1), counted.Streak().Days())
	assert.Equal(t, "2026-09-17", counted.Streak().LastDay().String())
}

func TestAWholeDayWithNoTakeStartsTheStreakAgain(t *testing.T) {
	counted := taken(beforeMidnight, beforeMidnight.Add(24*time.Hour), beforeMidnight.Add(3*24*time.Hour))

	assert.Equal(t, uint32(1), counted.Streak().Days())
	assert.Equal(t, uint32(2), counted.StreakBest(), "the best run is kept")
	assert.Equal(t, uint64(3), counted.TilesTaken())
}

func TestALateTakeCountsATileAndLeavesTheStreakAlone(t *testing.T) {
	counted := taken(beforeMidnight, beforeMidnight.Add(24*time.Hour), beforeMidnight.Add(-48*time.Hour))

	assert.Equal(t, uint64(3), counted.TilesTaken())
	assert.Equal(t, uint32(2), counted.Streak().Days())
	assert.Equal(t, "2026-09-18", counted.Streak().LastDay().String())
}

func TestAStreakCoversEachDayOfItsRunAndNoOther(t *testing.T) {
	counted := taken(beforeMidnight.Add(-24*time.Hour), beforeMidnight, beforeMidnight.Add(24*time.Hour))
	first, last := stats.DayOf(beforeMidnight.Add(-24*time.Hour)), stats.DayOf(beforeMidnight.Add(24*time.Hour))

	assert.True(t, counted.Streak().Covers(first))
	assert.True(t, counted.Streak().Covers(stats.DayOf(beforeMidnight)))
	assert.True(t, counted.Streak().Covers(last))
	assert.False(t, counted.Streak().Covers(last.Following()))
	assert.False(t, counted.Streak().Covers(stats.DayOf(beforeMidnight.Add(-48*time.Hour))))
	assert.False(t, stats.Stats{}.Streak().Covers(first), "no take covers no day")

	broken := taken(beforeMidnight, beforeMidnight.Add(3*24*time.Hour))
	assert.False(t, broken.Streak().Covers(stats.DayOf(beforeMidnight)), "a broken streak forgets its run")
}

func TestNoTakeIsNoDay(t *testing.T) {
	var counted stats.Stats

	assert.True(t, counted.Streak().LastDay().Empty())
	assert.Empty(t, counted.Streak().LastDay().String())
}

func TestAMessageCountsAMessageAndLeavesTheTilesAndTheStreakAlone(t *testing.T) {
	want := stats.StatsSnapshot{
		TilesTaken: 1, Streak: stats.StreakOf(1, stats.DayOf(beforeMidnight)), StreakBest: 1, DaysPlayed: 1, MessagesSent: 2,
		Counted: stats.Counted{TakesBelow: stats.TakeMark{Position: 0, Index: 1}, MessagesUntil: beforeMidnight.Add(time.Second)},
	}.Restore()

	assert.Equal(t, want, taken(beforeMidnight).WithMessage(beforeMidnight).WithMessage(beforeMidnight.Add(time.Second)))
	assert.True(t, stats.Stats{}.WithMessage(beforeMidnight).Streak().LastDay().Empty(),
		"a message is no take, so it starts no streak")
}

func TestEachCounterCountsItsOwnActAndLeavesTheStreakAlone(t *testing.T) {
	counted := stats.Stats{}.WithBomb(3).WithFortify(4, 137).WithFortify(9, 1).WithBonus(beforeMidnight).
		WithBonus(beforeMidnight.Add(time.Minute)).WithRightAnswer(beforeMidnight)

	assert.Equal(t, stats.StatsSnapshot{
		BombsDropped: 1, TilesFortified: 138, BonusesClaimed: 2, RightAnswers: 1,
		Counted: stats.Counted{
			BombsBelow: 4, FortifiesBelow: 10, BonusesUntil: beforeMidnight.Add(time.Minute), AnswersUntil: beforeMidnight,
		},
	}, counted.TakeSnapshot())
	assert.True(t, counted.Streak().LastDay().Empty(), "none of them is a take")
}

func TestALedgerFactAtOrBeforeTheLastOneCountedIsCountedAlready(t *testing.T) {
	counted := stats.Stats{}.WithTake(7, 0, beforeMidnight).WithTake(7, 0, beforeMidnight).WithTake(3, 0, beforeMidnight).
		WithBomb(7).WithBomb(7).WithFortify(7, 60).WithFortify(7, 60)

	assert.Equal(t, uint64(1), counted.TilesTaken())
	assert.Equal(t, uint64(1), counted.BombsDropped(), "takes, bombs and fortifies each keep their own mark")
	assert.Equal(t, uint64(60), counted.TilesFortified())
	assert.Equal(t, uint64(2), counted.WithTake(8, 0, beforeMidnight).TilesTaken())
}

func TestEachTakeOfASpreadCountsOnceThoughTheyShareAPosition(t *testing.T) {
	counted := stats.Stats{}
	for range 2 {
		for index := range uint32(7) {
			counted = counted.WithTake(7, index, beforeMidnight)
		}
	}

	assert.Equal(t, uint64(7), counted.TilesTaken())
	assert.Equal(t, stats.TakeMark{Position: 7, Index: 7}, counted.TakeSnapshot().Counted.TakesBelow)
}

func TestAFactAtOrBeforeTheTimeOfTheLastOneCountedIsCountedAlready(t *testing.T) {
	counted := stats.Stats{}.WithMessage(beforeMidnight).WithMessage(beforeMidnight).
		WithMessage(beforeMidnight.Add(-time.Second)).WithBonus(beforeMidnight).WithBonus(beforeMidnight).
		WithRightAnswer(beforeMidnight).WithRightAnswer(beforeMidnight)

	assert.Equal(t, uint64(1), counted.MessagesSent())
	assert.Equal(t, uint64(1), counted.BonusesClaimed())
	assert.Equal(t, uint64(1), counted.RightAnswers())
}

func TestAFactToldAgainWithMoreThanMicrosecondsIsTheOneKept(t *testing.T) {
	sent := beforeMidnight.Add(123_456_789 * time.Nanosecond)
	kept := stats.Stats{}.WithMessage(sent).TakeSnapshot()
	kept.Counted.MessagesUntil = kept.Counted.MessagesUntil.Truncate(time.Microsecond)

	assert.Equal(t, uint64(1), kept.Restore().WithMessage(sent).MessagesSent(), "postgres gave back the time it kept")
}

func TestTheMissionsDoneAreTheMostMissionsTellsItPaid(t *testing.T) {
	counted := stats.Stats{}.WithMissionsPaid(3).WithMissionsPaid(3).WithMissionsPaid(2)

	assert.Equal(t, uint64(3), counted.MissionsDone(), "a payment told twice or late counts nothing more")
	assert.Equal(t, uint64(12), counted.WithMissionsPaid(12).MissionsDone(), "the first one heard brings every mission before it")
	assert.Equal(t, stats.Streak{}, counted.Streak())
}

func TestEachKindOfMissionKeepsTheHighestTotalsMissionsTold(t *testing.T) {
	first := stats.Stats{}.WithMissionsDone(map[stats.MissionKind]stats.MissionTally{
		"take_for": {Done: 2, Goals: 360}, "bomb_in": {Done: 1, Goals: 2},
	})

	counted := first.WithMissionsDone(map[stats.MissionKind]stats.MissionTally{
		"take_for": {Done: 1, Goals: 180}, "catch_boxes": {Done: 1, Goals: 4},
	})

	assert.Equal(t, map[stats.MissionKind]stats.MissionTally{
		"take_for": {Done: 2, Goals: 360}, "bomb_in": {Done: 1, Goals: 2}, "catch_boxes": {Done: 1, Goals: 4},
	}, counted.MissionsByKind(), "a total told late lowers nothing, and a kind told alone keeps the others")
	assert.Equal(t, uint64(360), counted.MissionGoals("take_for"))
	assert.Zero(t, counted.MissionGoals("take_in"), "a kind never done")
	assert.Equal(t, counted, counted.WithMissionsDone(counted.MissionsByKind()), "told twice, it changes nothing")
	assert.Equal(t, map[stats.MissionKind]stats.MissionTally{"take_for": {Done: 2, Goals: 360}, "bomb_in": {Done: 1, Goals: 2}},
		first.MissionsByKind(), "the stats before the change are not changed")
}

func TestMissionsDoneTouchNeitherTheStreakNorTheMissionsPaid(t *testing.T) {
	counted := stats.Stats{}.WithMissionsDone(map[stats.MissionKind]stats.MissionTally{"take_for": {Done: 1, Goals: 180}})

	assert.Zero(t, counted.MissionsDone(), "the Missions track counts what was paid")
	assert.Equal(t, stats.Streak{}, counted.Streak())
	assert.Zero(t, counted.DaysPlayed())
	assert.Equal(t, stats.Stats{}, stats.Stats{}.WithMissionsDone(nil), "nothing told keeps no kind")
}

func TestATakeOnANewUTCDayAddsADayPlayed(t *testing.T) {
	assert.Equal(t, uint64(1), taken(beforeMidnight).DaysPlayed(), "the first take is the first day")
	assert.Equal(t, uint64(2), taken(beforeMidnight, beforeMidnight.Add(2*time.Second)).DaysPlayed(),
		"just past UTC midnight is the next day")
	assert.Equal(t, uint64(3), taken(beforeMidnight, beforeMidnight.Add(24*time.Hour), beforeMidnight.Add(5*24*time.Hour)).DaysPlayed(),
		"a day after a gap counts though the streak starts again")
}

func TestATakeOnTheSameDayOrALateOlderDayAddsNoDayPlayed(t *testing.T) {
	assert.Equal(t, uint64(1), taken(beforeMidnight.Add(-23*time.Hour), beforeMidnight).DaysPlayed())
	assert.Equal(t, uint64(2), taken(beforeMidnight, beforeMidnight.Add(24*time.Hour), beforeMidnight.Add(-48*time.Hour)).DaysPlayed(),
		"a late take older than the last day adds nothing, as for the streak")
}

func TestATakeCountedAlreadyAddsNoDayPlayed(t *testing.T) {
	counted := taken(beforeMidnight).WithTake(0, 0, beforeMidnight.Add(time.Hour))

	assert.Equal(t, uint64(1), counted.DaysPlayed())
}

func TestNoOtherFactAddsADayPlayed(t *testing.T) {
	counted := stats.Stats{}.WithMessage(beforeMidnight).WithBomb(1).WithFortify(2, 3).WithBonus(beforeMidnight).
		WithRightAnswer(beforeMidnight).WithMissionsPaid(4)

	assert.Zero(t, counted.DaysPlayed())
}

func day(t *testing.T, value string) stats.Day {
	t.Helper()

	parsed, err := stats.DayFrom(value)
	require.NoError(t, err)
	return parsed
}

func days(t *testing.T, values ...string) []stats.Day {
	t.Helper()

	parsed := make([]stats.Day, 0, len(values))
	for _, value := range values {
		parsed = append(parsed, day(t, value))
	}
	return parsed
}

func TestTheBackfillCountsEachLedgerDayUpToTheLastDayOnce(t *testing.T) {
	counted := stats.StatsSnapshot{Streak: stats.StreakOf(1, day(t, "2026-10-05"))}.Restore().
		WithPlayDays(days(t, "2026-09-20", "2026-09-25", "2026-09-25", "2026-10-05", "2026-10-07"))

	assert.Equal(t, uint64(3), counted.DaysPlayed(), "a day after the last one is the live count's")
}

func TestTheBackfillAlwaysCountsTheLastDay(t *testing.T) {
	counted := stats.StatsSnapshot{Streak: stats.StreakOf(1, day(t, "2026-10-05"))}.Restore().
		WithPlayDays(days(t, "2026-09-20", "2026-09-25"))

	assert.Equal(t, uint64(3), counted.DaysPlayed(), "a take flushed after the ledger was read")
}

func TestTheBestStreakIsTheFloorOfTheDaysPlayed(t *testing.T) {
	counted := stats.StatsSnapshot{Streak: stats.StreakOf(2, day(t, "2026-10-05")), StreakBest: 12}.Restore().
		WithPlayDays(days(t, "2026-10-04", "2026-10-05"))

	assert.Equal(t, uint64(12), counted.DaysPlayed(), "takes from before the ledger had accounts")
}

func TestTheBackfillNeverLowersTheDaysPlayed(t *testing.T) {
	counted := stats.StatsSnapshot{Streak: stats.StreakOf(1, day(t, "2026-10-05")), StreakBest: 1, DaysPlayed: 40}.Restore().
		WithPlayDays(days(t, "2026-10-05"))

	assert.Equal(t, uint64(40), counted.DaysPlayed())
}

func TestTheBackfillTwiceChangesNothing(t *testing.T) {
	ledger := days(t, "2026-09-20", "2026-09-25", "2026-10-05")
	once := stats.StatsSnapshot{Streak: stats.StreakOf(1, day(t, "2026-10-05")), StreakBest: 1}.Restore().WithPlayDays(ledger)

	assert.Equal(t, once, once.WithPlayDays(ledger))
}

func TestAnAccountWithNoTakeHasNoDayToBackfill(t *testing.T) {
	counted := stats.Stats{}.WithMessage(beforeMidnight).WithPlayDays(days(t, "2026-09-20"))

	assert.Zero(t, counted.DaysPlayed())
}

func TestADayReadsBackAsItIsWritten(t *testing.T) {
	assert.Equal(t, stats.DayOf(beforeMidnight), day(t, stats.DayOf(beforeMidnight).String()))

	_, err := stats.DayFrom("17/09/2026")
	require.Error(t, err)
}
