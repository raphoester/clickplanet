package missions_test

import (
	"fmt"
	"iter"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var (
	calendar = missions.NewCalendar(missions.Config{Cutoff: 21 * time.Hour})
	monday   = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	ada      = missions.AccountID{15: 1}
)

var world = missions.World{
	Territories: []missions.Territory{
		{Flag: "dz", Tiles: 90_000}, {Flag: "pw", Tiles: 70_000}, {Flag: "us", Tiles: 1_000}, {Flag: "fr", Tiles: 999},
	},
	CountrySizes: []missions.Holding{
		{Country: "br", Tiles: 9_000}, {Country: "fr", Tiles: 3_000}, {Country: "is", Tiles: 2_999}, {Country: "va", Tiles: 1},
	},
}

// What a mission aims at, whatever its goal.
func targetOf(order missions.Order) string {
	switch order := order.(type) {
	case missions.TakeFor:
		return "for " + order.Flag
	case missions.TakeFrom:
		return "from " + order.Flag
	case missions.TakeIn:
		return "in " + order.Ground
	}
	return fmt.Sprintf("%T", order)
}

func within(t *testing.T, goal, least, most uint32, msg any) {
	t.Helper()

	assert.GreaterOrEqual(t, goal, least, msg)
	assert.LessOrEqual(t, goal, most, msg)
	assert.Zero(t, goal%10, "%v: a goal is a round number", msg)
}

type drawn struct {
	flag  string
	day   int
	brief []missions.MissionSnapshot
}

func (d drawn) String() string { return fmt.Sprintf("%s, day %d", d.flag, d.day) }

// The briefs of four flags over as many days, from the test world.
func briefs(days int) iter.Seq[drawn] {
	return func(yield func(drawn) bool) {
		for _, flag := range []string{"fr", "br", "is", "dz"} {
			for day := range days {
				brief := missions.Draw(calendar.DayOf(monday.AddDate(0, 0, day)), flag, world).Snapshot()
				if !yield(drawn{flag: flag, day: day, brief: brief}) {
					return
				}
			}
		}
	}
}

var position uint64

func take(flag, previous, ground string) missions.Take {
	position++
	return missions.Take{Account: ada, Flag: flag, Previous: previous, Ground: ground, At: monday, Position: position}
}

func TestADayEndsAtTheCutoff(t *testing.T) {
	before := calendar.DayOf(time.Date(2026, 10, 26, 20, 59, 59, 0, time.UTC))
	at := calendar.DayOf(time.Date(2026, 10, 26, 21, 0, 0, 0, time.UTC))

	assert.Equal(t, time.Date(2026, 10, 26, 21, 0, 0, 0, time.UTC), before.EndsAt())
	assert.Equal(t, time.Date(2026, 10, 27, 21, 0, 0, 0, time.UTC), at.EndsAt(), "the cutoff starts the next day")
	assert.Equal(t, "2026-10-27", at.String(), "a day is named by the date it ends on")
}

func TestADayWithNoCutoffIsAUTCDay(t *testing.T) {
	day := missions.NewCalendar(missions.Config{}).DayOf(monday)

	assert.Equal(t, time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC), day.EndsAt())
}

func TestACutoffIsATimeOfDay(t *testing.T) {
	require.NoError(t, missions.Config{Cutoff: 21 * time.Hour}.Validate())
	require.Error(t, missions.Config{Cutoff: 24 * time.Hour}.Validate())
	require.Error(t, missions.Config{Cutoff: -time.Minute}.Validate())
}

func TestTheFirstMissionIsTwoToFourBanksForTheFlag(t *testing.T) {
	goals := cpcolls.NewSet[uint32]()
	for d := range briefs(30) {
		order, ok := d.brief[0].Order.(missions.TakeFor)
		require.True(t, ok, "%s: %T", d, d.brief[0].Order)
		assert.Equal(t, d.flag, order.Flag, d)
		within(t, order.Goal, 120, 240, d)
		goals.Add(order.Goal)
	}

	assert.Greater(t, goals.Len(), 1, "the goal changes with the day")
}

func TestTheSecondMissionTakesFromARivalHoldingAThousandTiles(t *testing.T) {
	for d := range briefs(30) {
		order, ok := d.brief[1].Order.(missions.TakeFrom)
		require.True(t, ok, "%s: %T", d, d.brief[1].Order)
		assert.Contains(t, []string{"dz", "pw", "us"}, order.Flag, "%s: no rival under 1,000 tiles", d)
		assert.NotEqual(t, d.flag, order.Flag, d)
		within(t, order.Goal, 80, 160, d)
	}
}

func TestTheRivalIsDrawnByTheGroundItHolds(t *testing.T) {
	shares := missions.World{Territories: []missions.Territory{
		{Flag: "dz", Tiles: 60_000}, {Flag: "pw", Tiles: 30_000}, {Flag: "us", Tiles: 10_000},
	}}
	const days = 1_000

	drawn := map[string]int{}
	for day := range days {
		order, ok := missions.Draw(calendar.DayOf(monday.AddDate(0, 0, day)), "fr", shares).Snapshot()[1].Order.(missions.TakeFrom)
		require.True(t, ok, "day %d", day)
		drawn[order.Flag]++
	}

	assert.InDelta(t, 0.6, float64(drawn["dz"])/days, 0.05, "the leader, drawn %d days of %d", drawn["dz"], days)
	assert.InDelta(t, 0.3, float64(drawn["pw"])/days, 0.05, "the runner-up, drawn %d days of %d", drawn["pw"], days)
	assert.InDelta(t, 0.1, float64(drawn["us"])/days, 0.05, "the smallest, drawn %d days of %d", drawn["us"], days)
}

func TestWithNoRivalBigEnoughTheSecondMissionIsForTheFlag(t *testing.T) {
	empty := missions.World{Territories: []missions.Territory{{Flag: "de", Tiles: 999}}}

	second := missions.Draw(calendar.DayOf(monday), "fr", empty).Snapshot()[1]

	order, ok := second.Order.(missions.TakeFor)
	require.True(t, ok, "%T", second.Order)
	assert.Equal(t, "fr", order.Flag)
	within(t, order.Goal, 160, 320, "twice the rival's goal")
}

func TestTheThirdMissionIsABigGroundThatIsNotTheFlags(t *testing.T) {
	for d := range briefs(30) {
		order, ok := d.brief[2].Order.(missions.TakeIn)
		require.True(t, ok, "%s: %T", d, d.brief[2].Order)
		assert.Contains(t, []string{"br", "fr"}, order.Ground, "%s: no ground under 3,000 tiles", d)
		assert.NotEqual(t, d.flag, order.Ground, d)
		within(t, order.Goal, 300, 600, d)
	}
}

func TestAGroundTheFlagHoldsAllButAThousandTilesOfIsNoTarget(t *testing.T) {
	held := missions.World{
		Territories: []missions.Territory{
			{Flag: "fr", Tiles: 10_001, Grounds: []missions.Holding{{Country: "in", Tiles: 2_001}, {Country: "br", Tiles: 8_000}}},
		},
		CountrySizes: []missions.Holding{{Country: "in", Tiles: 3_000}, {Country: "br", Tiles: 9_000}},
	}

	grounds := map[string]*cpcolls.Set[string]{"fr": cpcolls.NewSet[string](), "de": cpcolls.NewSet[string]()}
	for flag, targets := range grounds {
		for day := range 30 {
			for _, mission := range missions.Draw(calendar.DayOf(monday.AddDate(0, 0, day)), flag, held).Snapshot() {
				if order, ok := mission.Order.(missions.TakeIn); ok {
					targets.Add(order.Ground)
				}
			}
		}
	}

	assert.Equal(t, cpcolls.NewSet("br"), grounds["fr"], "999 tiles of India are not France's, 1,000 of Brazil are")
	assert.Equal(t, cpcolls.NewSet("br", "in"), grounds["de"], "Germany holds none of either")
}

func TestWithNoGroundBigEnoughTheThirdMissionIsForTheFlag(t *testing.T) {
	small := missions.World{CountrySizes: []missions.Holding{{Country: "is", Tiles: 2_999}}}

	third := missions.Draw(calendar.DayOf(monday), "fr", small).Snapshot()[2]

	order, ok := third.Order.(missions.TakeFor)
	require.True(t, ok, "%T", third.Order)
	assert.Equal(t, "fr", order.Flag)
	within(t, order.Goal, 300, 600, "the ground's goal, for the flag")
}

func TestADayHasThreeToFiveMissions(t *testing.T) {
	counts := cpcolls.NewSet[int]()
	for d := range briefs(30) {
		assert.GreaterOrEqual(t, len(d.brief), 3, d)
		assert.LessOrEqual(t, len(d.brief), 5, d)
		counts.Add(len(d.brief))
	}

	assert.Equal(t, cpcolls.NewSet(3, 4, 5), counts, "the count changes with the day")
}

func TestTheExtraMissionsAreMoreRivalsAndGroundsNeverAimedAtTwice(t *testing.T) {
	for d := range briefs(30) {
		targets := cpcolls.NewSet[string]()
		for _, mission := range d.brief {
			assert.False(t, targets.Contains(targetOf(mission.Order)), "%s: %s twice", d, targetOf(mission.Order))
			targets.Add(targetOf(mission.Order))
		}
		for _, extra := range d.brief[3:] {
			switch order := extra.Order.(type) {
			case missions.TakeFrom:
				within(t, order.Goal, 80, 160, d)
			case missions.TakeIn:
				within(t, order.Goal, 300, 600, d)
			default:
				assert.Failf(t, "an extra is a rival or a ground", "%s: %T", d, extra.Order)
			}
		}
	}
}

func TestWithNothingMoreToAimAtADayHasThreeMissions(t *testing.T) {
	bare := missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}

	for day := range 30 {
		assert.Len(t, missions.Draw(calendar.DayOf(monday.AddDate(0, 0, day)), "fr", bare).Snapshot(), 3, "day %d", day)
	}
}

var purses = map[missions.RewardKind][2]uint32{
	missions.RewardShields:       {15, 30},
	missions.RewardSpreadClicks:  {4, 8},
	missions.RewardBomb:          {1, 1},
	missions.RewardEncloseClicks: {2, 3},
}

func TestAMissionPaysAsMuchOfAChargeAsItsPurseHolds(t *testing.T) {
	amounts := map[missions.RewardKind]*cpcolls.Set[uint32]{}
	for kind := range purses {
		amounts[kind] = cpcolls.NewSet[uint32]()
	}
	for d := range briefs(60) {
		for _, mission := range d.brief {
			purse, known := purses[mission.Reward.Kind]
			require.True(t, known, "%s pays %s: never a refill", d, mission.Reward.Kind)
			assert.GreaterOrEqual(t, mission.Reward.Amount, purse[0], "%s: %+v", d, mission.Reward)
			assert.LessOrEqual(t, mission.Reward.Amount, purse[1], "%s: %+v", d, mission.Reward)
			amounts[mission.Reward.Kind].Add(mission.Reward.Amount)
		}
	}

	for kind, purse := range purses {
		assert.Equal(t, int(purse[1]-purse[0]+1), amounts[kind].Len(), "%s pays every amount of its purse over 240 briefs", kind)
	}
}

func TestADayPaysEveryChargeOnceBeforeAnyTwice(t *testing.T) {
	for d := range briefs(30) {
		kinds := cpcolls.NewSet[missions.RewardKind]()
		for _, mission := range d.brief {
			kinds.Add(mission.Reward.Kind)
		}

		assert.Equal(t, min(len(d.brief), len(purses)), kinds.Len(), d)
	}
}

func TestEachMissionPaysEveryChargeOnSomeDay(t *testing.T) {
	kinds := make([]*cpcolls.Set[missions.RewardKind], 3)
	for n := range kinds {
		kinds[n] = cpcolls.NewSet[missions.RewardKind]()
	}
	for d := range briefs(30) {
		for n, mission := range d.brief[:len(kinds)] {
			kinds[n].Add(mission.Reward.Kind)
		}
	}

	for n, drawn := range kinds {
		assert.Equal(t, len(purses), drawn.Len(), "mission %d pays only some of the charges over 120 briefs", n+1)
	}
}

func TestTheSameDayAndFlagDrawTheSameBrief(t *testing.T) {
	day := calendar.DayOf(monday)

	first, again := missions.Draw(day, "fr", world), missions.Draw(day, "fr", world)

	assert.Equal(t, first.Snapshot(), again.Snapshot())
}

func TestEachMissionDrawnHasAnIDOfItsOwn(t *testing.T) {
	seen := cpcolls.NewSet[missions.MissionID]()
	for _, flag := range []string{"fr", "dz"} {
		for days := range 2 {
			for _, mission := range missions.Draw(calendar.DayOf(monday.AddDate(0, 0, days)), flag, world).Snapshot() {
				assert.False(t, seen.Contains(mission.ID), "%s, day %d", flag, days)
				seen.Add(mission.ID)
			}
		}
	}
}

func TestABriefReadsBackFromItsSnapshot(t *testing.T) {
	drawn := missions.Draw(calendar.DayOf(monday), "fr", world)

	restored, err := missions.BriefOf(drawn.Snapshot())

	require.NoError(t, err)
	assert.Equal(t, drawn.Snapshot(), restored.Snapshot())
}

func TestAMissionWithNoOrderIsRefused(t *testing.T) {
	_, err := missions.BriefOf([]missions.MissionSnapshot{{ID: bank}})

	assert.ErrorIs(t, err, missions.ErrNoOrder)
}

func briefOf(t *testing.T, snapshots ...missions.MissionSnapshot) missions.Brief {
	t.Helper()

	brief, err := missions.BriefOf(snapshots)
	require.NoError(t, err)
	return brief
}

var (
	bank   = missions.MissionID{15: 1}
	rival  = missions.MissionID{15: 2}
	ground = missions.MissionID{15: 3}
)

var frBrief = []missions.MissionSnapshot{
	{ID: bank, Order: missions.TakeFor{Flag: "fr", Goal: 2}},
	{ID: rival, Order: missions.TakeFrom{Flag: "dz", Goal: 5}},
	{ID: ground, Order: missions.TakeIn{Ground: "br", Goal: 5}},
}

func TestTheFirstTakeOfTheDayEnlistsItsFlag(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{})
	require.False(t, sheet.Enlisted())

	first := take("fr", "", "es")
	sheet = sheet.WithTake(first, briefOf(t, frBrief...))

	assert.True(t, sheet.Enlisted())
	assert.Equal(t, missions.SheetSnapshot{
		Flag: "fr", Progress: map[missions.MissionID]uint32{bank: 1}, Paid: cpcolls.NewSet[missions.MissionID](),
		TakesBelow: first.Position + 1,
	}, sheet.Snapshot())
}

func TestATakeCountsForEveryMissionItFits(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{}).WithTake(take("fr", "dz", "br"), briefOf(t, frBrief...))

	assert.Equal(t, map[missions.MissionID]uint32{bank: 1, rival: 1, ground: 1}, sheet.Snapshot().Progress)
}

func TestATakeForAnotherFlagThanTheDaysCountsNothing(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{Flag: "fr"})

	sheet = sheet.WithTake(take("de", "dz", "br"), briefOf(
		t,
		missions.MissionSnapshot{ID: bank, Order: missions.TakeFor{Flag: "de", Goal: 60}},
		missions.MissionSnapshot{ID: rival, Order: missions.TakeFrom{Flag: "dz", Goal: 60}},
	))

	assert.Equal(t, "fr", sheet.Snapshot().Flag, "the day's flag is kept")
	assert.Empty(t, sheet.Snapshot().Progress)
}

func TestAMissionCountsNoFurtherThanItsGoal(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{})
	for range 5 {
		sheet = sheet.WithTake(take("fr", "", ""), briefOf(t, frBrief...))
	}

	assert.Equal(t, uint32(2), sheet.Snapshot().Progress[bank])
}

func TestATakeToldTwiceCountsOnce(t *testing.T) {
	brief := briefOf(t, frBrief...)
	told := take("fr", "dz", "")

	sheet := missions.SheetOf(missions.SheetSnapshot{}).WithTake(told, brief).WithTake(told, brief)

	assert.Equal(t, map[missions.MissionID]uint32{bank: 1, rival: 1}, sheet.Snapshot().Progress)
	assert.Equal(t, told.Position+1, sheet.Snapshot().TakesBelow)
}

func TestATakeBelowTheMarkCountsNothing(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{Flag: "fr", TakesBelow: 1_000})

	sheet = sheet.WithTake(missions.Take{Account: ada, Flag: "fr", At: monday, Position: 999}, briefOf(t, frBrief...))

	assert.Empty(t, sheet.Snapshot().Progress)
}

func TestATakeThatCountsNothingLeavesTheMark(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{Flag: "fr", TakesBelow: 5})

	sheet = sheet.WithTake(missions.Take{Account: ada, Flag: "de", At: monday, Position: 9}, briefOf(t, frBrief...))

	assert.Equal(t, uint64(5), sheet.Snapshot().TakesBelow)
}

func TestAWithTakeLeavesTheSheetItWasCalledOnAlone(t *testing.T) {
	before := missions.SheetOf(missions.SheetSnapshot{})

	_ = before.WithTake(take("fr", "", ""), briefOf(t, frBrief...))

	assert.False(t, before.Enlisted())
	assert.Empty(t, before.Snapshot().Progress)
}

func TestEachMissionDoneIsPaidOnce(t *testing.T) {
	shields := missions.Reward{Kind: missions.RewardShields, Amount: 10}
	bomb := missions.Reward{Kind: missions.RewardBomb, Amount: 1}
	brief := briefOf(
		t,
		missions.MissionSnapshot{ID: bank, Order: missions.TakeFor{Flag: "fr", Goal: 2}, Reward: shields},
		missions.MissionSnapshot{ID: rival, Order: missions.TakeFrom{Flag: "dz", Goal: 5}, Reward: bomb},
		missions.MissionSnapshot{ID: ground, Order: missions.TakeIn{Ground: "br", Goal: 5}, Reward: bomb},
	)
	sheet := missions.SheetOf(missions.SheetSnapshot{Flag: "fr", Progress: map[missions.MissionID]uint32{bank: 2, rival: 5, ground: 4}})

	paid, payments := sheet.WithPayments(brief)
	_, again := paid.WithPayments(brief)

	assert.Equal(t, []missions.Payment{{Mission: bank, Reward: shields}, {Mission: rival, Reward: bomb}}, payments,
		"4 of 5 is not done")
	assert.Empty(t, again)
	assert.Equal(t, cpcolls.NewSet(bank, rival), paid.Snapshot().Paid)
	assert.True(t, sheet.Snapshot().Paid.Empty(), "the sheet it was called on is left alone")
}

func TestEachMissionSaysHowFarItWentAndWhetherItIsDoneAndPaid(t *testing.T) {
	brief := briefOf(t, frBrief...)
	sheet := missions.SheetOf(missions.SheetSnapshot{
		Flag: "fr", Progress: map[missions.MissionID]uint32{bank: 2, rival: 3}, Paid: cpcolls.NewSet(bank),
	})

	assert.Equal(t, []missions.MissionStatus{
		{Mission: frBrief[0], Progress: 2, Done: true, Paid: true},
		{Mission: frBrief[1], Progress: 3},
		{Mission: frBrief[2]},
	}, sheet.Statuses(brief))
}
