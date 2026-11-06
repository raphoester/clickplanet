package flagmissions_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/flagmissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var monday = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

var world = missions.World{
	Territories:  []missions.Territory{{Flag: "dz", Tiles: 60_000}, {Flag: "pw", Tiles: 30_000}, {Flag: "us", Tiles: 10_000}},
	CountrySizes: []missions.Holding{{Country: "br", Tiles: 12_000}, {Country: "in", Tiles: 4_000}, {Country: "is", Tiles: 2_999}},
}

func dayOf(t *testing.T, snapshot flagmissions.DaySnapshot) flagmissions.Day {
	t.Helper()

	day, err := flagmissions.DayOf(snapshot)
	require.NoError(t, err)
	return day
}

func statusOf(t *testing.T, day flagmissions.Day) flagmissions.Status {
	t.Helper()

	status, ok := day.Status()
	require.True(t, ok, "the day has a mission")
	return status
}

func drawn(t *testing.T, count int, flag string, yesterday flagmissions.Day) []flagmissions.Status {
	t.Helper()

	statuses := make([]flagmissions.Status, 0, count)
	for n := range count {
		statuses = append(statuses, statusOf(t, flagmissions.Draw(days.TestingDayAt(monday.AddDate(0, 0, n)), flag, world, yesterday, flagmissions.Played)))
	}
	return statuses
}

func TestADayHasOneMissionOfEitherKind(t *testing.T) {
	kinds := cpcolls.NewSet[string]()
	for _, status := range drawn(t, 60, "fr", flagmissions.Day{}) {
		kinds.Add(fmt.Sprintf("%T", status.Mission.Order))
	}

	assert.Equal(t, cpcolls.NewSet("flagmissions.TakeIn", "flagmissions.BreakShields"), kinds)
}

func TestTheSmallerTheFlagBesideTheBiggestTheMoreItsMissionIsWorth(t *testing.T) {
	want := map[string]uint32{"dz": 3, "pw": 6, "us": 9, "fr": 10}

	for flag, points := range want {
		for _, status := range drawn(t, 10, flag, flagmissions.Day{}) {
			assert.Equal(t, points, status.Mission.Points, "%s: 60,000 tiles is the biggest, 30,000 half of it, 10,000 a sixth, none", flag)
		}
	}
}

func TestWithNoTileHeldAnywhereAMissionIsWorthTheMost(t *testing.T) {
	blank := missions.World{CountrySizes: world.CountrySizes, Territories: []missions.Territory{{Flag: "dz"}}}

	status := statusOf(t, flagmissions.Draw(days.TestingDayAt(monday), "fr", blank, flagmissions.Day{}, flagmissions.Played))

	assert.Equal(t, uint32(10), status.Mission.Points)
}

func TestAConquestIsOnABigGroundThatIsNotTheFlags(t *testing.T) {
	for _, status := range drawn(t, 60, "br", flagmissions.Day{}) {
		if order, ok := status.Mission.Order.(flagmissions.TakeIn); ok {
			assert.Equal(t, "in", order.Ground, "Iceland is under 3,000 tiles, and Brazil is the flag's own")
		}
	}
}

func groundsOf(t *testing.T, world missions.World, yesterday flagmissions.Day) *cpcolls.Set[string] {
	t.Helper()

	grounds := cpcolls.NewSet[string]()
	for n := range 60 {
		status := statusOf(t, flagmissions.Draw(days.TestingDayAt(monday.AddDate(0, 0, n)), "fr", world, yesterday, flagmissions.Played))
		if order, ok := status.Mission.Order.(flagmissions.TakeIn); ok {
			grounds.Add(order.Ground)
		}
	}
	return grounds
}

func TestAConquestIsOnAGroundWithRoomForItsGoalBesideTheTilesTheFlagHoldsThere(t *testing.T) {
	held := missions.World{
		Territories:  []missions.Territory{{Flag: "fr", Tiles: 5_000, Grounds: []missions.Holding{{Country: "cn", Tiles: 5_000}}}},
		CountrySizes: []missions.Holding{{Country: "br", Tiles: 12_000}, {Country: "cn", Tiles: 9_000}, {Country: "sd", Tiles: 3_536}},
	}

	assert.Equal(t, cpcolls.NewSet("br"), groundsOf(t, held, dayOf(t, flagmissions.DaySnapshot{Takes: 10_005})),
		"a goal of 5,000: Sudan has 3,536 tiles, and France holds 5,000 of China's 9,000")
	assert.Equal(t, cpcolls.NewSet("br", "cn", "sd"), groundsOf(t, held, flagmissions.Day{}),
		"a goal of 100 fits on each")
}

func TestWithNoGroundRoomyEnoughTheGoalIsTheMostRoomThereIs(t *testing.T) {
	held := missions.World{
		Territories:  []missions.Territory{{Flag: "fr", Tiles: 1_000, Grounds: []missions.Holding{{Country: "sd", Tiles: 1_000}}}},
		CountrySizes: []missions.Holding{{Country: "dz", Tiles: 3_568}, {Country: "sd", Tiles: 3_536}},
	}
	busy := dayOf(t, flagmissions.DaySnapshot{Takes: 10_005})

	for n := range 30 {
		status := statusOf(t, flagmissions.Draw(days.TestingDayAt(monday.AddDate(0, 0, n)), "fr", held, busy, flagmissions.Played))
		assert.Equal(t, flagmissions.TakeIn{Ground: "dz", Goal: 3_560}, status.Mission.Order,
			"France holds 1,000 of Sudan's 3,536 tiles, none of Algeria's 3,568, rounded down to 10")
	}
}

func TestTheShieldsToBreakAreARivalsDrawnByTheGroundItHolds(t *testing.T) {
	broken := map[string]int{}
	for _, status := range drawn(t, 2_000, "fr", flagmissions.Day{}) {
		if order, ok := status.Mission.Order.(flagmissions.BreakShields); ok {
			broken[order.Flag]++
		}
	}
	total := broken["dz"] + broken["pw"] + broken["us"]

	assert.InDelta(t, 0.6, float64(broken["dz"])/float64(total), 0.05, "the leader, %d of %d", broken["dz"], total)
	assert.InDelta(t, 0.3, float64(broken["pw"])/float64(total), 0.05, "the runner-up, %d of %d", broken["pw"], total)
	assert.InDelta(t, 0.1, float64(broken["us"])/float64(total), 0.05, "the smallest, %d of %d", broken["us"], total)
}

func goalOf(order flagmissions.Order) uint32 {
	switch order := order.(type) {
	case flagmissions.TakeIn:
		return order.Goal
	case flagmissions.BreakShields:
		return order.Goal
	}
	return 0
}

func goalsOf(t *testing.T, yesterday flagmissions.Day) map[string]uint32 {
	t.Helper()

	goals := map[string]uint32{}
	for _, status := range drawn(t, 60, "fr", yesterday) {
		goals[fmt.Sprintf("%T", status.Mission.Order)] = goalOf(status.Mission.Order)
	}
	return goals
}

func TestTheGoalIsHalfOfWhatTheFlagDidYesterday(t *testing.T) {
	goals := goalsOf(t, dayOf(t, flagmissions.DaySnapshot{Takes: 10_005, Strikes: 1_009}))

	assert.Equal(t, map[string]uint32{"flagmissions.TakeIn": 5_000, "flagmissions.BreakShields": 500}, goals)
}

func TestASmallFlagsGoalIsHalfOfItsSmallDay(t *testing.T) {
	goals := goalsOf(t, dayOf(t, flagmissions.DaySnapshot{Takes: 356, Strikes: 181}))

	assert.Equal(t, map[string]uint32{"flagmissions.TakeIn": 170, "flagmissions.BreakShields": 90}, goals)
}

func TestAFlagThatDidLittleYesterdayHasTheLeastGoal(t *testing.T) {
	goals := goalsOf(t, dayOf(t, flagmissions.DaySnapshot{Takes: 40, Strikes: 3}))

	assert.Equal(t, map[string]uint32{"flagmissions.TakeIn": 100, "flagmissions.BreakShields": 30}, goals)
}

func TestWithNothingToAimAtTheDayHasNoMission(t *testing.T) {
	_, ok := flagmissions.Draw(days.TestingDayAt(monday), "fr", missions.World{}, flagmissions.Day{}, flagmissions.Played).Status()

	assert.False(t, ok)
}

func TestTheSameDayAndFlagDrawTheSameMissionAndEachHasAnIDOfItsOwn(t *testing.T) {
	day := days.TestingDayAt(monday)

	first, again := flagmissions.Draw(day, "fr", world, flagmissions.Day{}, flagmissions.Played), flagmissions.Draw(day, "fr", world, flagmissions.Day{}, flagmissions.Played)
	assert.Equal(t, first.Snapshot(), again.Snapshot())
	ids := cpcolls.NewSet[flagmissions.MissionID]()
	for _, flag := range []string{"fr", "de"} {
		for n := range 2 {
			ids.Add(statusOf(t, flagmissions.Draw(days.TestingDayAt(monday.AddDate(0, 0, n)), flag, world, flagmissions.Day{}, flagmissions.Played)).Mission.ID)
		}
	}
	assert.Equal(t, 4, ids.Len())
}

var (
	mission = flagmissions.MissionID{15: 7}
	conquer = &flagmissions.MissionSnapshot{ID: mission, Order: flagmissions.TakeIn{Ground: "br", Goal: 2}, Points: 3}
	shields = &flagmissions.MissionSnapshot{ID: mission, Order: flagmissions.BreakShields{Flag: "dz", Goal: 2}, Points: 1}
)

var position uint64

func take(ground string) flagmissions.Take {
	position++
	return flagmissions.Take{Flag: "fr", Ground: ground, At: monday, Position: position}
}

func strike(owner string) flagmissions.Strike {
	position++
	return flagmissions.Strike{Flag: "fr", Owner: owner, At: monday, Position: position}
}

func TestEveryTakeCountsForTomorrowAndTheOnesOnTheGroundOnTheMission(t *testing.T) {
	day := dayOf(t, flagmissions.DaySnapshot{Mission: conquer})

	day, _ = day.WithTake(take("es"))
	day, done := day.WithTake(take("br"))

	assert.Empty(t, done)
	assert.Equal(t, uint64(2), day.Snapshot().Takes)
	assert.Equal(t, uint32(1), day.Snapshot().Progress)
}

func TestATakeToldTwiceCountsOnce(t *testing.T) {
	told := take("br")

	day, _ := dayOf(t, flagmissions.DaySnapshot{Mission: conquer}).WithTake(told)
	day, _ = day.WithTake(told)

	assert.Equal(t, uint64(1), day.Snapshot().Takes)
	assert.Equal(t, uint32(1), day.Snapshot().Progress)
	assert.Equal(t, missions.LedgerMark{Position: told.Position, Index: 1}, day.Snapshot().TakesBelow)
}

func TestEachTakeOfASpreadCountsThoughTheyShareAPosition(t *testing.T) {
	first := take("br")
	second := first
	second.Index = 1

	day, _ := dayOf(t, flagmissions.DaySnapshot{Mission: conquer}).WithTake(first)
	day, done := day.WithTake(second)
	day, _ = day.WithTake(first)

	assert.Equal(t, uint64(2), day.Snapshot().Takes)
	assert.Equal(t, []flagmissions.Done{{Mission: *conquer, Progress: 2}}, done)
}

func TestTheTakeThatFinishesTheMissionGivesItsPointsOnce(t *testing.T) {
	day := dayOf(t, flagmissions.DaySnapshot{Mission: conquer, Progress: 1})

	day, done := day.WithTake(take("br"))
	day, again := day.WithTake(take("br"))

	assert.Equal(t, []flagmissions.Done{{Mission: *conquer, Progress: 2}}, done)
	assert.Empty(t, again)
	assert.True(t, day.Snapshot().Done)
	assert.Equal(t, uint32(2), day.Snapshot().Progress, "no further than the goal")
}

func TestOnlyTheRivalsShieldsCountAndEachStrikeForTomorrow(t *testing.T) {
	day := dayOf(t, flagmissions.DaySnapshot{Mission: shields})

	day, _ = day.WithStrike(strike("pw"))
	day, _ = day.WithStrike(strike("dz"))
	day, done := day.WithStrike(strike("dz"))

	assert.Equal(t, []flagmissions.Done{{Mission: *shields, Progress: 2}}, done)
	assert.Equal(t, uint64(3), day.Snapshot().Strikes)
	assert.Equal(t, missions.LedgerMark{Position: position, Index: 1}, day.Snapshot().StrikesBelow)
}

func TestAStrikeToldTwiceCountsOnce(t *testing.T) {
	told := strike("dz")

	day, _ := dayOf(t, flagmissions.DaySnapshot{Mission: shields}).WithStrike(told)
	day, _ = day.WithStrike(told)

	assert.Equal(t, uint64(1), day.Snapshot().Strikes)
	assert.Equal(t, uint32(1), day.Snapshot().Progress)
}

func TestEachStrikeOfASpreadOrABombCountsThoughTheyShareAPosition(t *testing.T) {
	first := strike("dz")
	second := first
	second.Index = 1

	day, _ := dayOf(t, flagmissions.DaySnapshot{Mission: shields}).WithStrike(first)
	day, done := day.WithStrike(second)
	day, _ = day.WithStrike(first)

	assert.Equal(t, uint64(2), day.Snapshot().Strikes)
	assert.Equal(t, []flagmissions.Done{{Mission: *shields, Progress: 2}}, done)
}

func TestABombOnTheFlagsOwnShieldsBreaksNothing(t *testing.T) {
	own := strike("fr")

	day, _ := dayOf(t, flagmissions.DaySnapshot{Mission: shields}).WithStrike(own)

	assert.Zero(t, day.Snapshot().Strikes)
	assert.Equal(t, missions.LedgerMark{Position: own.Position, Index: 1}, day.Snapshot().StrikesBelow, "still counted once")
}

func TestATakeMovesNoShieldsAndAStrikeNoGround(t *testing.T) {
	onGround, _ := dayOf(t, flagmissions.DaySnapshot{Mission: shields}).WithTake(take("br"))
	onShields, _ := dayOf(t, flagmissions.DaySnapshot{Mission: conquer}).WithStrike(strike("dz"))

	assert.Zero(t, onGround.Snapshot().Progress)
	assert.Zero(t, onShields.Snapshot().Progress)
}

func TestADayWithNoMissionStillCountsForTomorrow(t *testing.T) {
	taken, struck := take("br"), strike("dz")
	day, done := flagmissions.Day{}.WithTake(taken)
	day, _ = day.WithStrike(struck)

	assert.Empty(t, done)
	assert.Equal(t, flagmissions.DaySnapshot{
		Takes: 1, TakesBelow: missions.LedgerMark{Position: taken.Position, Index: 1},
		Strikes: 1, StrikesBelow: missions.LedgerMark{Position: struck.Position, Index: 1},
	}, day.Snapshot())
}

func TestAMissionWithNoOrderIsRefused(t *testing.T) {
	_, err := flagmissions.DayOf(flagmissions.DaySnapshot{Mission: &flagmissions.MissionSnapshot{ID: mission}})

	assert.ErrorIs(t, err, flagmissions.ErrNoOrder)
}

func TestAnotherSizingDrawsItsOwnGoalsAndPoints(t *testing.T) {
	sizing := flagmissions.Sizing{GoalPercent: 20, LeastConquest: 500, LeastShields: 200, FewestPoints: 2, MostPoints: 5}
	busy := dayOf(t, flagmissions.DaySnapshot{Takes: 10_005, Strikes: 400})

	for n := range 30 {
		status := statusOf(t, flagmissions.Draw(days.TestingDayAt(monday.AddDate(0, 0, n)), "pw", world, busy, sizing))
		assert.Equal(t, uint32(3), status.Mission.Points, "half the biggest flag: 5 - 3/2, 1.5 rounded to 2")
		switch order := status.Mission.Order.(type) {
		case flagmissions.TakeIn:
			assert.Equal(t, uint32(2_000), order.Goal, "20% of 10,005 tiles, rounded down to 10")
		case flagmissions.BreakShields:
			assert.Equal(t, uint32(200), order.Goal, "20% of 400 shields is under the least")
		default:
			t.Fatalf("not a kind of flag mission: %T", order)
		}
	}
}

func TestASizingAsksAPartOfTheDayAndGivesAtLeastAPoint(t *testing.T) {
	require.NoError(t, flagmissions.Played.Validate())
	for _, sizing := range []flagmissions.Sizing{
		{GoalPercent: 0, FewestPoints: 3, MostPoints: 10},
		{GoalPercent: 50, FewestPoints: 0, MostPoints: 10},
		{GoalPercent: 50, FewestPoints: 6, MostPoints: 5},
	} {
		assert.ErrorIs(t, sizing.Validate(), flagmissions.ErrInvalidSizing, "%+v", sizing)
	}
}
