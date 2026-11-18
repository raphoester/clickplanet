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

func TestTheBackfillNeverLowersTheDaysPlayed(t *testing.T) {
	counted := stats.StatsSnapshot{Streak: stats.StreakOf(1, day(t, "2026-10-05")), StreakBest: 1, DaysPlayed: 40}.Restore().
		WithPlayDays(days(t, "2026-10-05"))

	assert.Equal(t, uint64(40), counted.DaysPlayed())
}

func TestAnAccountWithNoTakeHasNoDayToBackfill(t *testing.T) {
	counted := stats.Stats{}.WithMessage(beforeMidnight).WithPlayDays(days(t, "2026-09-20"))

	assert.Zero(t, counted.DaysPlayed())
}

func dayRange(t *testing.T, first, last string) []stats.Day {
	t.Helper()

	var run []stats.Day
	for d := day(t, first); !day(t, last).Before(d); d = d.Following() {
		run = append(run, d)
	}
	return run
}

// The stats counted live from a take at noon on each day, and the ledger that holds them.
func playedOn(t *testing.T, ledger ...[]stats.Day) (stats.Stats, []stats.Day) {
	t.Helper()

	var (
		counted stats.Stats
		played  []stats.Day
	)
	for _, days := range ledger {
		played = append(played, days...)
	}
	for position, d := range played {
		noon, err := time.Parse(time.DateTime, d.String()+" 12:00:00")
		require.NoError(t, err)
		counted = counted.WithTake(uint64(position), 0, noon)
	}
	return counted, played
}

func TestAForgivenGapJoinsTheRunBeforeItToTheStreak(t *testing.T) {
	counted, ledger := playedOn(t, days(t, "2026-09-30"), dayRange(t, "2026-10-09", "2026-11-14"), dayRange(t, "2026-11-16", "2026-11-17"))
	require.Equal(t, stats.StreakOf(2, day(t, "2026-11-17")), counted.Streak())
	require.Equal(t, uint32(37), counted.StreakBest())

	forgiven, err := counted.WithForgivenDay(day(t, "2026-11-15"), ledger, day(t, "2026-11-17"))

	require.NoError(t, err)
	assert.Equal(t, stats.StreakOf(40, day(t, "2026-11-17")), forgiven.Streak())
	assert.Equal(t, uint32(40), forgiven.StreakBest())
	assert.Equal(t, days(t, "2026-11-15"), forgiven.ForgivenDays())
	want := counted.TakeSnapshot()
	want.Streak, want.StreakBest, want.ForgivenDays = forgiven.Streak(), 40, days(t, "2026-11-15")
	assert.Equal(t, want, forgiven.TakeSnapshot(), "no day played, no tile, nothing else")
	assert.Equal(t, uint64(40), forgiven.DaysPlayed())
}

func TestAForgivenYesterdayIsTheLastDaySoTheNextTakeExtendsTheStreak(t *testing.T) {
	counted, ledger := playedOn(t, dayRange(t, "2026-11-10", "2026-11-15"))

	forgiven, err := counted.WithForgivenDay(day(t, "2026-11-16"), ledger, day(t, "2026-11-17"))

	require.NoError(t, err)
	assert.Equal(t, stats.StreakOf(7, day(t, "2026-11-16")), forgiven.Streak())
	assert.Equal(t, uint32(7), forgiven.StreakBest())
	assert.Equal(t, uint64(6), forgiven.DaysPlayed())

	next := forgiven.WithTake(100, 0, time.Date(2026, 11, 17, 8, 0, 0, 0, time.UTC))
	assert.Equal(t, stats.StreakOf(8, day(t, "2026-11-17")), next.Streak())
	assert.Equal(t, uint64(7), next.DaysPlayed(), "the take is a day played, the forgiven day is not")
}

func TestAForgivenDayBeforeTheRunLengthensIt(t *testing.T) {
	counted, ledger := playedOn(t, dayRange(t, "2026-11-01", "2026-11-03"), dayRange(t, "2026-11-05", "2026-11-06"))

	forgiven, err := counted.WithForgivenDay(day(t, "2026-11-04"), ledger, day(t, "2026-11-17"))

	require.NoError(t, err)
	assert.Equal(t, stats.StreakOf(6, day(t, "2026-11-06")), forgiven.Streak(), "a broken streak keeps its last day")
}

func TestTheDaysForgivenBeforeStayOnTheStreak(t *testing.T) {
	counted, ledger := playedOn(t, dayRange(t, "2026-11-01", "2026-11-02"), dayRange(t, "2026-11-04", "2026-11-05"),
		dayRange(t, "2026-11-07", "2026-11-08"))
	once, err := counted.WithForgivenDay(day(t, "2026-11-06"), ledger, day(t, "2026-11-17"))
	require.NoError(t, err)

	twice, err := once.WithForgivenDay(day(t, "2026-11-03"), ledger, day(t, "2026-11-17"))

	require.NoError(t, err)
	assert.Equal(t, stats.StreakOf(8, day(t, "2026-11-08")), twice.Streak())
	assert.Equal(t, days(t, "2026-11-03", "2026-11-06"), twice.ForgivenDays())
}

func TestABetterRunBeforeKeepsTheBestStreak(t *testing.T) {
	counted, ledger := playedOn(t, dayRange(t, "2026-10-01", "2026-10-20"), dayRange(t, "2026-11-01", "2026-11-02"),
		dayRange(t, "2026-11-04", "2026-11-05"))

	forgiven, err := counted.WithForgivenDay(day(t, "2026-11-03"), ledger, day(t, "2026-11-17"))

	require.NoError(t, err)
	assert.Equal(t, uint32(5), forgiven.Streak().Days())
	assert.Equal(t, uint32(20), forgiven.StreakBest())
}

func TestTheStreakKeptIsAFloorOfTheRun(t *testing.T) {
	counted := stats.StatsSnapshot{Streak: stats.StreakOf(9, day(t, "2026-09-18")), StreakBest: 9, DaysPlayed: 9}.Restore()

	forgiven, err := counted.WithForgivenDay(day(t, "2026-09-19"), days(t, "2026-09-17", "2026-09-18"), day(t, "2026-11-17"))

	require.NoError(t, err)
	assert.Equal(t, stats.StreakOf(10, day(t, "2026-09-19")), forgiven.Streak(), "the ledger holds no account before 2026-09-16")
}

func TestADayThatCannotBeForgivenIsRefusedAndChangesNothing(t *testing.T) {
	counted, ledger := playedOn(t, dayRange(t, "2026-11-01", "2026-11-03"), dayRange(t, "2026-11-05", "2026-11-06"),
		dayRange(t, "2026-11-10", "2026-11-11"))
	forgiven, err := counted.WithForgivenDay(day(t, "2026-11-09"), ledger, day(t, "2026-11-12"))
	require.NoError(t, err)

	for name, refused := range map[string]struct {
		day   string
		today string
		err   error
	}{
		"today":                       {day: "2026-11-12", today: "2026-11-12", err: stats.ErrNotPast},
		"a day to come":               {day: "2026-11-13", today: "2026-11-12", err: stats.ErrNotPast},
		"a day played on the run":     {day: "2026-11-10", today: "2026-11-12", err: stats.ErrDayPlayed},
		"a day played before the run": {day: "2026-11-02", today: "2026-11-12", err: stats.ErrDayPlayed},
		"a day forgiven already":      {day: "2026-11-09", today: "2026-11-12", err: stats.ErrDayForgiven},
		"two days after the last day": {day: "2026-11-13", today: "2026-11-20", err: stats.ErrJoinsNothing},
		"a gap apart from the run":    {day: "2026-11-04", today: "2026-11-12", err: stats.ErrJoinsNothing},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := forgiven.WithForgivenDay(day(t, refused.day), ledger, day(t, refused.today))

			require.ErrorIs(t, err, refused.err)
			assert.Equal(t, forgiven, got)
			assert.Equal(t, forgiven, stats.ForgivenDay(day(t, refused.day), ledger, day(t, refused.today))(forgiven),
				"the change leaves the stats as they are")
		})
	}
}

func TestAnAccountWithNoTakeHasNoStreakToForgive(t *testing.T) {
	counted := stats.Stats{}.WithMessage(beforeMidnight)

	_, err := counted.WithForgivenDay(day(t, "2026-09-16"), nil, day(t, "2026-09-18"))

	require.ErrorIs(t, err, stats.ErrNoStreak)
}

func TestABackfillCountsNoForgivenDay(t *testing.T) {
	counted, ledger := playedOn(t, dayRange(t, "2026-11-10", "2026-11-12"), dayRange(t, "2026-11-14", "2026-11-15"))
	gap, err := counted.WithForgivenDay(day(t, "2026-11-13"), ledger, day(t, "2026-11-17"))
	require.NoError(t, err)
	yesterday, err := gap.WithForgivenDay(day(t, "2026-11-16"), ledger, day(t, "2026-11-17"))
	require.NoError(t, err)
	require.Equal(t, uint32(7), yesterday.StreakBest())

	backfilled := yesterday.WithPlayDays(ledger)

	assert.Equal(t, uint64(5), backfilled.DaysPlayed(), "neither the gap nor the last day, forgiven, nor the best streak")
}

func TestTheBestStreakLessTheForgivenDaysIsTheFloor(t *testing.T) {
	counted := stats.StatsSnapshot{
		Streak: stats.StreakOf(12, day(t, "2026-10-05")), StreakBest: 12, ForgivenDays: days(t, "2026-09-30"),
	}.Restore().WithPlayDays(days(t, "2026-10-04", "2026-10-05"))

	assert.Equal(t, uint64(11), counted.DaysPlayed())
}

func TestTheForgivenDaysAreKeptInOrderOnce(t *testing.T) {
	kept := stats.StatsSnapshot{ForgivenDays: days(t, "2026-11-05", "2026-11-01", "2026-11-05")}.Restore()

	assert.Equal(t, days(t, "2026-11-01", "2026-11-05"), kept.TakeSnapshot().ForgivenDays)
	assert.Nil(t, stats.StatsSnapshot{ForgivenDays: []stats.Day{}}.Restore().ForgivenDays(), "none is no day")
}

func TestADayReadsBackAsItIsWritten(t *testing.T) {
	assert.Equal(t, stats.DayOf(beforeMidnight), day(t, stats.DayOf(beforeMidnight).String()))

	_, err := stats.DayFrom("17/09/2026")
	require.Error(t, err)
}
