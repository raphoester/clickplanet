package players_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

var beforeMidnight = time.Date(2026, 9, 17, 23, 59, 59, 0, time.UTC)

func taken(at ...time.Time) players.Stats {
	var stats players.Stats
	for position, take := range at {
		stats = stats.WithTake(uint64(position), 0, take)
	}
	return stats
}

func TestTheFirstTakeStartsAStreakOfOne(t *testing.T) {
	stats := taken(beforeMidnight)

	assert.Equal(t, players.StatsSnapshot{
		TilesTaken: 1, Streak: players.StreakOf(1, players.DayOf(beforeMidnight)), StreakBest: 1,
		Counted: players.Counted{TakesBelow: players.TakeMark{Position: 0, Index: 1}},
	}.Restore(), stats)
}

func TestATakeOnTheSameDayCountsATileAndNotADay(t *testing.T) {
	stats := taken(beforeMidnight.Add(-23*time.Hour), beforeMidnight)

	assert.Equal(t, uint64(2), stats.TilesTaken())
	assert.Equal(t, uint32(1), stats.Streak().Days())
}

func TestATakeJustAfterUTCMidnightExtendsTheStreak(t *testing.T) {
	stats := taken(beforeMidnight, beforeMidnight.Add(2*time.Second))

	assert.Equal(t, uint32(2), stats.Streak().Days())
	assert.Equal(t, uint32(2), stats.StreakBest())
	assert.Equal(t, "2026-09-18", stats.Streak().LastDay().String())
}

func TestTheDayIsUTCWhateverTheZoneOfTheTake(t *testing.T) {
	paris := time.FixedZone("CEST", 2*60*60)
	stats := taken(beforeMidnight, time.Date(2026, 9, 18, 0, 30, 0, 0, paris))

	assert.Equal(t, uint32(1), stats.Streak().Days())
	assert.Equal(t, "2026-09-17", stats.Streak().LastDay().String())
}

func TestAWholeDayWithNoTakeStartsTheStreakAgain(t *testing.T) {
	stats := taken(beforeMidnight, beforeMidnight.Add(24*time.Hour), beforeMidnight.Add(3*24*time.Hour))

	assert.Equal(t, uint32(1), stats.Streak().Days())
	assert.Equal(t, uint32(2), stats.StreakBest(), "the best run is kept")
	assert.Equal(t, uint64(3), stats.TilesTaken())
}

func TestALateTakeCountsATileAndLeavesTheStreakAlone(t *testing.T) {
	stats := taken(beforeMidnight, beforeMidnight.Add(24*time.Hour), beforeMidnight.Add(-48*time.Hour))

	assert.Equal(t, uint64(3), stats.TilesTaken())
	assert.Equal(t, uint32(2), stats.Streak().Days())
	assert.Equal(t, "2026-09-18", stats.Streak().LastDay().String())
}

func TestAStreakReadsZeroOnceAWholeDayWentBy(t *testing.T) {
	stats := taken(beforeMidnight.Add(-24*time.Hour), beforeMidnight)
	today := players.DayOf(beforeMidnight)

	assert.Equal(t, uint32(2), stats.AsOf(today).Streak().Days())
	assert.Equal(t, uint32(2), stats.AsOf(today.Following()).Streak().Days(), "yesterday's streak can still be extended today")
	assert.Equal(t, uint32(0), stats.AsOf(today.Following().Following()).Streak().Days())
	assert.Equal(t, uint32(2), stats.AsOf(today.Following().Following()).StreakBest())
}

func TestNoTakeIsNoDay(t *testing.T) {
	var stats players.Stats

	assert.True(t, stats.Streak().LastDay().Empty())
	assert.Empty(t, stats.Streak().LastDay().String())
	assert.Equal(t, stats, stats.AsOf(players.DayOf(beforeMidnight)))
}

func TestAMessageCountsAMessageAndLeavesTheTilesAndTheStreakAlone(t *testing.T) {
	want := players.StatsSnapshot{
		TilesTaken: 1, Streak: players.StreakOf(1, players.DayOf(beforeMidnight)), StreakBest: 1, MessagesSent: 2,
		Counted: players.Counted{TakesBelow: players.TakeMark{Position: 0, Index: 1}, MessagesUntil: beforeMidnight.Add(time.Second)},
	}.Restore()

	assert.Equal(t, want, taken(beforeMidnight).WithMessage(beforeMidnight).WithMessage(beforeMidnight.Add(time.Second)))
	assert.True(t, players.Stats{}.WithMessage(beforeMidnight).Streak().LastDay().Empty(),
		"a message is no take, so it starts no streak")
}

func TestEachCounterCountsItsOwnActAndLeavesTheStreakAlone(t *testing.T) {
	stats := players.Stats{}.WithBomb(3).WithFortify(4, 137).WithFortify(9, 1).WithBonus(beforeMidnight).
		WithBonus(beforeMidnight.Add(time.Minute)).WithRightAnswer(beforeMidnight)

	assert.Equal(t, players.StatsSnapshot{
		BombsDropped: 1, TilesFortified: 138, BonusesClaimed: 2, RightAnswers: 1,
		Counted: players.Counted{
			BombsBelow: 4, FortifiesBelow: 10, BonusesUntil: beforeMidnight.Add(time.Minute), AnswersUntil: beforeMidnight,
		},
	}, stats.TakeSnapshot())
	assert.True(t, stats.Streak().LastDay().Empty(), "none of them is a take")
}

func TestALedgerFactAtOrBeforeTheLastOneCountedIsCountedAlready(t *testing.T) {
	stats := players.Stats{}.WithTake(7, 0, beforeMidnight).WithTake(7, 0, beforeMidnight).WithTake(3, 0, beforeMidnight).
		WithBomb(7).WithBomb(7).WithFortify(7, 60).WithFortify(7, 60)

	assert.Equal(t, uint64(1), stats.TilesTaken())
	assert.Equal(t, uint64(1), stats.BombsDropped(), "takes, bombs and fortifies each keep their own mark")
	assert.Equal(t, uint64(60), stats.TilesFortified())
	assert.Equal(t, uint64(2), stats.WithTake(8, 0, beforeMidnight).TilesTaken())
}

func TestEachTakeOfASpreadCountsOnceThoughTheyShareAPosition(t *testing.T) {
	stats := players.Stats{}
	for range 2 {
		for index := range uint32(7) {
			stats = stats.WithTake(7, index, beforeMidnight)
		}
	}

	assert.Equal(t, uint64(7), stats.TilesTaken())
	assert.Equal(t, players.TakeMark{Position: 7, Index: 7}, stats.TakeSnapshot().Counted.TakesBelow)
}

func TestAFactAtOrBeforeTheTimeOfTheLastOneCountedIsCountedAlready(t *testing.T) {
	stats := players.Stats{}.WithMessage(beforeMidnight).WithMessage(beforeMidnight).
		WithMessage(beforeMidnight.Add(-time.Second)).WithBonus(beforeMidnight).WithBonus(beforeMidnight).
		WithRightAnswer(beforeMidnight).WithRightAnswer(beforeMidnight)

	assert.Equal(t, uint64(1), stats.MessagesSent())
	assert.Equal(t, uint64(1), stats.BonusesClaimed())
	assert.Equal(t, uint64(1), stats.RightAnswers())
}

func TestAFactToldAgainWithMoreThanMicrosecondsIsTheOneKept(t *testing.T) {
	sent := beforeMidnight.Add(123_456_789 * time.Nanosecond)
	kept := players.Stats{}.WithMessage(sent).TakeSnapshot()
	kept.Counted.MessagesUntil = kept.Counted.MessagesUntil.Truncate(time.Microsecond)

	assert.Equal(t, uint64(1), kept.Restore().WithMessage(sent).MessagesSent(), "postgres gave back the time it kept")
}

func TestTheMissionsDoneAreTheMostMissionsTellsItPaid(t *testing.T) {
	stats := players.Stats{}.WithMissionsPaid(3).WithMissionsPaid(3).WithMissionsPaid(2)

	assert.Equal(t, uint64(3), stats.MissionsDone(), "a payment told twice or late counts nothing more")
	assert.Equal(t, uint64(12), stats.WithMissionsPaid(12).MissionsDone(), "the first one heard brings every mission before it")
	assert.Equal(t, players.Streak{}, stats.Streak())
}
