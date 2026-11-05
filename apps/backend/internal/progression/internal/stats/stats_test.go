package stats_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
		TilesTaken: 1, Streak: stats.StreakOf(1, stats.DayOf(beforeMidnight)), StreakBest: 1,
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

func TestAStreakReadsZeroOnceAWholeDayWentBy(t *testing.T) {
	counted := taken(beforeMidnight.Add(-24*time.Hour), beforeMidnight)
	today := stats.DayOf(beforeMidnight)

	assert.Equal(t, uint32(2), counted.AsOf(today).Streak().Days())
	assert.Equal(t, uint32(2), counted.AsOf(today.Following()).Streak().Days(), "yesterday's streak can still be extended today")
	assert.Equal(t, uint32(0), counted.AsOf(today.Following().Following()).Streak().Days())
	assert.Equal(t, uint32(2), counted.AsOf(today.Following().Following()).StreakBest())
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
	assert.Equal(t, counted, counted.AsOf(stats.DayOf(beforeMidnight)))
}

func TestAMessageCountsAMessageAndLeavesTheTilesAndTheStreakAlone(t *testing.T) {
	want := stats.StatsSnapshot{
		TilesTaken: 1, Streak: stats.StreakOf(1, stats.DayOf(beforeMidnight)), StreakBest: 1, MessagesSent: 2,
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
