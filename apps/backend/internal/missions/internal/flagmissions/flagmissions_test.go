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
	CountrySizes: []missions.Holding{{Country: "br", Tiles: 9_000}, {Country: "in", Tiles: 4_000}, {Country: "is", Tiles: 2_999}},
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
		statuses = append(statuses, statusOf(t, flagmissions.Draw(days.TestingDayAt(monday.AddDate(0, 0, n)), flag, world, yesterday)))
	}
	return statuses
}

func TestADayHasOneMissionOfEitherKindWorthOneToThreePoints(t *testing.T) {
	points, kinds := cpcolls.NewSet[uint32](), cpcolls.NewSet[string]()
	for _, status := range drawn(t, 60, "fr", flagmissions.Day{}) {
		points.Add(status.Mission.Points)
		kinds.Add(fmt.Sprintf("%T", status.Mission.Order))
	}

	assert.Equal(t, cpcolls.NewSet[uint32](1, 2, 3), points)
	assert.Equal(t, cpcolls.NewSet("flagmissions.TakeIn", "flagmissions.BreakShields"), kinds)
}

func TestAConquestIsOnABigGroundThatIsNotTheFlags(t *testing.T) {
	for _, status := range drawn(t, 60, "br", flagmissions.Day{}) {
		if order, ok := status.Mission.Order.(flagmissions.TakeIn); ok {
			assert.Equal(t, "in", order.Ground, "Iceland is under 3,000 tiles, and Brazil is the flag's own")
		}
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

func TestTheGoalIsAShareOfWhatTheFlagDidYesterdayByItsPoints(t *testing.T) {
	busy := dayOf(t, flagmissions.DaySnapshot{Takes: 10_005, Strikes: 1_009})
	want := map[string]map[uint32]uint32{
		"flagmissions.TakeIn":       {1: 5_000, 2: 7_500, 3: 10_000},
		"flagmissions.BreakShields": {1: 500, 2: 750, 3: 1_000},
	}

	for _, status := range drawn(t, 60, "fr", busy) {
		kind := fmt.Sprintf("%T", status.Mission.Order)
		assert.Equal(t, want[kind][status.Mission.Points], goalOf(status.Mission.Order), "%s worth %d", kind, status.Mission.Points)
	}
}

func TestAFlagThatDidLittleYesterdayHasTheLeastGoal(t *testing.T) {
	want := map[string]map[uint32]uint32{
		"flagmissions.TakeIn":       {1: 1_000, 2: 1_500, 3: 2_000},
		"flagmissions.BreakShields": {1: 300, 2: 450, 3: 600},
	}

	for _, status := range drawn(t, 60, "fr", dayOf(t, flagmissions.DaySnapshot{Takes: 40, Strikes: 3})) {
		kind := fmt.Sprintf("%T", status.Mission.Order)
		assert.Equal(t, want[kind][status.Mission.Points], goalOf(status.Mission.Order), "%s worth %d", kind, status.Mission.Points)
	}
}

func TestWithNothingToAimAtTheDayHasNoMission(t *testing.T) {
	_, ok := flagmissions.Draw(days.TestingDayAt(monday), "fr", missions.World{}, flagmissions.Day{}).Status()

	assert.False(t, ok)
}

func TestTheSameDayAndFlagDrawTheSameMissionAndEachHasAnIDOfItsOwn(t *testing.T) {
	day := days.TestingDayAt(monday)

	first, again := flagmissions.Draw(day, "fr", world, flagmissions.Day{}), flagmissions.Draw(day, "fr", world, flagmissions.Day{})
	assert.Equal(t, first.Snapshot(), again.Snapshot())
	ids := cpcolls.NewSet[flagmissions.MissionID]()
	for _, flag := range []string{"fr", "de"} {
		for n := range 2 {
			ids.Add(statusOf(t, flagmissions.Draw(days.TestingDayAt(monday.AddDate(0, 0, n)), flag, world, flagmissions.Day{})).Mission.ID)
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
	assert.Equal(t, []flagmissions.Done{{Mission: mission, Points: 3}}, done)
}

func TestTheTakeThatFinishesTheMissionGivesItsPointsOnce(t *testing.T) {
	day := dayOf(t, flagmissions.DaySnapshot{Mission: conquer, Progress: 1})

	day, done := day.WithTake(take("br"))
	day, again := day.WithTake(take("br"))

	assert.Equal(t, []flagmissions.Done{{Mission: mission, Points: 3}}, done)
	assert.Empty(t, again)
	assert.True(t, day.Snapshot().Done)
	assert.Equal(t, uint32(2), day.Snapshot().Progress, "no further than the goal")
}

func TestOnlyTheRivalsShieldsCountAndEachStrikeForTomorrow(t *testing.T) {
	day := dayOf(t, flagmissions.DaySnapshot{Mission: shields})

	day, _ = day.WithStrike(strike("pw"))
	day, _ = day.WithStrike(strike("dz"))
	day, done := day.WithStrike(strike("dz"))

	assert.Equal(t, []flagmissions.Done{{Mission: mission, Points: 1}}, done)
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
	assert.Equal(t, []flagmissions.Done{{Mission: mission, Points: 1}}, done)
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
