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
	for _, take := range at {
		stats = stats.WithTake(take)
	}
	return stats
}

func TestTheFirstTakeStartsAStreakOfOne(t *testing.T) {
	stats := taken(beforeMidnight)

	assert.Equal(t, players.StatsOf(players.AccountID{}, 1, players.StreakOf(1, players.DayOf(beforeMidnight)), 1, 0), stats)
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
	want := players.StatsOf(players.AccountID{}, 1, players.StreakOf(1, players.DayOf(beforeMidnight)), 1, 2)

	assert.Equal(t, want, taken(beforeMidnight).WithMessage().WithMessage())
	assert.True(t, players.Stats{}.WithMessage().Streak().LastDay().Empty(), "a message is no take, so it starts no streak")
}
