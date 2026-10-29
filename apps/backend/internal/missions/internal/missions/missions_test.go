package missions_test

import (
	"fmt"
	"iter"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var (
	monday = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	ada    = missions.AccountID{15: 1}
	bob    = missions.AccountID{15: 2}
)

var world = missions.World{
	Territories: []missions.Territory{
		{Flag: "dz", Tiles: 90_000, Grounds: []missions.Holding{{Country: "br", Tiles: 4_000}, {Country: "fr", Tiles: 299}}},
		{Flag: "pw", Tiles: 70_000},
		{Flag: "us", Tiles: 1_000},
		{Flag: "fr", Tiles: 999},
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
	case missions.BombIn:
		return "bomb " + order.Ground
	}
	return fmt.Sprintf("%T", order)
}

func within(t *testing.T, goal, least, most uint32, msg any) {
	t.Helper()

	between(t, goal, least, most, msg)
	assert.Zero(t, goal%10, "%v: a goal is a round number", msg)
}

func between(t *testing.T, goal, least, most uint32, msg any) {
	t.Helper()

	assert.GreaterOrEqual(t, goal, least, msg)
	assert.LessOrEqual(t, goal, most, msg)
}

type drawn struct {
	flag  string
	day   int
	brief []missions.MissionSnapshot
}

func (d drawn) String() string { return fmt.Sprintf("%s, day %d", d.flag, d.day) }

// The briefs of four players of four flags over as many days, from the test world.
func briefs(count int) iter.Seq[drawn] {
	return func(yield func(drawn) bool) {
		for n, flag := range []string{"fr", "br", "is", "dz"} {
			for day := range count {
				brief := missions.Draw(days.TestingDayAt(monday.AddDate(0, 0, day)), missions.AccountID{15: byte(n + 1)}, flag, world).Snapshot()
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
	return missions.Take{Flag: flag, Previous: previous, Ground: ground, Position: position}
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
	const draws = 1_000

	drawn := map[string]int{}
	for day := range draws {
		order, ok := missions.Draw(days.TestingDayAt(monday.AddDate(0, 0, day)), ada, "fr", shares).Snapshot()[1].Order.(missions.TakeFrom)
		require.True(t, ok, "day %d", day)
		drawn[order.Flag]++
	}

	assert.InDelta(t, 0.6, float64(drawn["dz"])/draws, 0.05, "the leader, drawn %d days of %d", drawn["dz"], draws)
	assert.InDelta(t, 0.3, float64(drawn["pw"])/draws, 0.05, "the runner-up, drawn %d days of %d", drawn["pw"], draws)
	assert.InDelta(t, 0.1, float64(drawn["us"])/draws, 0.05, "the smallest, drawn %d days of %d", drawn["us"], draws)
}

func TestWithNoRivalBigEnoughTheSecondMissionIsForTheFlag(t *testing.T) {
	empty := missions.World{Territories: []missions.Territory{{Flag: "de", Tiles: 999}}}

	second := missions.Draw(days.TestingDayAt(monday), ada, "fr", empty).Snapshot()[1]

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
			for _, mission := range missions.Draw(days.TestingDayAt(monday.AddDate(0, 0, day)), ada, flag, held).Snapshot() {
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

	third := missions.Draw(days.TestingDayAt(monday), ada, "fr", small).Snapshot()[2]

	order, ok := third.Order.(missions.TakeFor)
	require.True(t, ok, "%T", third.Order)
	assert.Equal(t, "fr", order.Flag)
	within(t, order.Goal, 300, 600, "the ground's goal, for the flag")
}

func TestADayHasFourToFiveMissions(t *testing.T) {
	counts := cpcolls.NewSet[int]()
	for d := range briefs(30) {
		counts.Add(len(d.brief))
	}

	assert.Equal(t, cpcolls.NewSet(4, 5), counts, "the count changes with the day")
}

func TestAfterTheBankTheRivalAndTheGroundComeFeatsOfOtherKindsEachOnce(t *testing.T) {
	kinds := cpcolls.NewSet[string]()
	for d := range briefs(60) {
		targets := cpcolls.NewSet[string]()
		for _, mission := range d.brief {
			assert.False(t, targets.Contains(targetOf(mission.Order)), "%s: %s twice", d, targetOf(mission.Order))
			targets.Add(targetOf(mission.Order))
		}
		for _, extra := range d.brief[3:] {
			switch order := extra.Order.(type) {
			case missions.TakeFromIn:
				assert.Equal(t, missions.TakeFromIn{Flag: "dz", Ground: "br", Goal: order.Goal}, order, "%s: dz holds 300 tiles in br alone", d)
				within(t, order.Goal, 40, 80, d)
			case missions.CatchBoxes:
				between(t, order.Goal, 3, 6, d)
			case missions.AnswerQuizzes:
				between(t, order.Goal, 2, 4, d)
			case missions.BombIn:
				assert.Contains(t, []string{"br", "fr"}, order.Ground, "%s: a big ground", d)
				assert.NotEqual(t, d.flag, order.Ground, d)
				between(t, order.Goal, 1, 2, d)
			case missions.ShieldTiles:
				assert.Equal(t, uint32(10), order.Shields, d)
				between(t, order.Goal, 1, 3, d)
			case missions.FortifyTiles:
				within(t, order.Goal, 20, 60, d)
			default:
				assert.Failf(t, "an extra is a feat", "%s: %T", d, extra.Order)
			}
			kinds.Add(fmt.Sprintf("%T", extra.Order))
		}
	}

	assert.Equal(t, 6, kinds.Len(), "every kind of feat over 240 briefs: %v", kinds)
}

func TestARaidIsNeverOnTheFlagItself(t *testing.T) {
	for d := range briefs(30) {
		for _, mission := range d.brief {
			if raid, ok := mission.Order.(missions.TakeFromIn); ok {
				assert.NotEqual(t, d.flag, raid.Flag, d)
			}
		}
	}
}

func TestWithNothingToAimAtADayStillHasItsFeats(t *testing.T) {
	bare := missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}

	for day := range 30 {
		brief := missions.Draw(days.TestingDayAt(monday.AddDate(0, 0, day)), ada, "fr", bare).Snapshot()
		assert.GreaterOrEqual(t, len(brief), 4, "day %d", day)
		for _, extra := range brief[3:] {
			assert.NotContains(t, []string{"missions.BombIn", "missions.TakeFromIn"}, fmt.Sprintf("%T", extra.Order),
				"day %d: no ground to bomb, no rival's ground to raid", day)
		}
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

func TestTheSameDayAndAccountDrawTheSameBrief(t *testing.T) {
	day := days.TestingDayAt(monday)

	first, again := missions.Draw(day, ada, "fr", world), missions.Draw(day, ada, "fr", world)

	assert.Equal(t, first.Snapshot(), again.Snapshot())
}

func TestTwoPlayersOfAFlagDrawTheirOwnDays(t *testing.T) {
	differ := 0
	for day := range 30 {
		at := days.TestingDayAt(monday.AddDate(0, 0, day))
		adas, bobs := missions.Draw(at, ada, "fr", world).Snapshot(), missions.Draw(at, bob, "fr", world).Snapshot()
		for n := range min(len(adas), len(bobs)) {
			if adas[n].Order != bobs[n].Order || adas[n].Reward != bobs[n].Reward {
				differ++
				break
			}
		}
	}

	assert.Greater(t, differ, 25, "of 30 days")
}

func TestEachMissionDrawnHasAnIDOfItsOwn(t *testing.T) {
	seen := cpcolls.NewSet[missions.MissionID]()
	for _, account := range []missions.AccountID{ada, bob} {
		for later := range 2 {
			for _, mission := range missions.Draw(days.TestingDayAt(monday.AddDate(0, 0, later)), account, "fr", world).Snapshot() {
				assert.False(t, seen.Contains(mission.ID), "%s, day %d", account, later)
				seen.Add(mission.ID)
			}
		}
	}
}

func TestABriefReadsBackFromItsSnapshot(t *testing.T) {
	drawn := missions.Draw(days.TestingDayAt(monday), ada, "fr", world)

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

func TestATakeCountsForEveryMissionItFits(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{}).WithFact(take("fr", "dz", "br"), briefOf(t, frBrief...))

	assert.Equal(t, map[missions.MissionID]uint32{bank: 1, rival: 1, ground: 1}, sheet.Snapshot().Progress)
}

func TestATakeForAnotherFlagCountsForTheMissionsItFits(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{}).WithFact(take("de", "dz", "br"), briefOf(t, frBrief...))

	assert.Equal(t, map[missions.MissionID]uint32{rival: 1, ground: 1}, sheet.Snapshot().Progress, "the tiles for France do not")
}

func TestAMissionCountsNoFurtherThanItsGoal(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{})
	for range 5 {
		sheet = sheet.WithFact(take("fr", "", ""), briefOf(t, frBrief...))
	}

	assert.Equal(t, uint32(2), sheet.Snapshot().Progress[bank])
}

func TestATakeToldTwiceCountsOnce(t *testing.T) {
	brief := briefOf(t, frBrief...)
	told := take("fr", "dz", "")

	sheet := missions.SheetOf(missions.SheetSnapshot{}).WithFact(told, brief).WithFact(told, brief)

	assert.Equal(t, map[missions.MissionID]uint32{bank: 1, rival: 1}, sheet.Snapshot().Progress)
	assert.Equal(t, missions.LedgerMark{Position: told.Position, Index: 1}, sheet.Snapshot().Counted.TakesBelow)
}

func TestEachTakeOfASpreadCountsOnceThoughTheyShareAPosition(t *testing.T) {
	brief := briefOf(t, frBrief...)
	sheet := missions.SheetOf(missions.SheetSnapshot{})
	for range 2 {
		for index := range uint32(2) {
			sheet = sheet.WithFact(missions.Take{Flag: "fr", Position: 7, Index: index}, brief)
		}
	}

	assert.Equal(t, uint32(2), sheet.Snapshot().Progress[bank])
	assert.Equal(t, missions.LedgerMark{Position: 7, Index: 2}, sheet.Snapshot().Counted.TakesBelow)
}

func TestATakeBelowTheMarkCountsNothing(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{Counted: missions.Counted{TakesBelow: missions.LedgerMark{Position: 1_000}}})

	sheet = sheet.WithFact(missions.Take{Flag: "fr", Position: 999}, briefOf(t, frBrief...))

	assert.Empty(t, sheet.Snapshot().Progress)
}

func TestATakeThatCountsNothingLeavesTheMark(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{Counted: missions.Counted{TakesBelow: missions.LedgerMark{Position: 5}}})

	sheet = sheet.WithFact(missions.Take{Flag: "de", Position: 9}, briefOf(t, frBrief...))

	assert.Equal(t, missions.LedgerMark{Position: 5}, sheet.Snapshot().Counted.TakesBelow)
}

func TestAWithFactLeavesTheSheetItWasCalledOnAlone(t *testing.T) {
	before := missions.SheetOf(missions.SheetSnapshot{})

	_ = before.WithFact(take("fr", "", ""), briefOf(t, frBrief...))

	assert.Empty(t, before.Snapshot().Progress)
}

var (
	raid    = missions.MissionID{15: 4}
	boxes   = missions.MissionID{15: 5}
	answers = missions.MissionID{15: 6}
	bombs   = missions.MissionID{15: 7}
	walls   = missions.MissionID{15: 8}
	fortify = missions.MissionID{15: 9}
)

var featBrief = []missions.MissionSnapshot{
	{ID: raid, Order: missions.TakeFromIn{Flag: "dz", Ground: "br", Goal: 5}},
	{ID: boxes, Order: missions.CatchBoxes{Goal: 5}},
	{ID: answers, Order: missions.AnswerQuizzes{Goal: 5}},
	{ID: bombs, Order: missions.BombIn{Ground: "br", Goal: 5}},
	{ID: walls, Order: missions.ShieldTiles{Shields: 10, Goal: 5}},
	{ID: fortify, Order: missions.FortifyTiles{Goal: 50}},
}

func TestEachFactCountsOnTheMissionsOfItsKind(t *testing.T) {
	for _, tc := range []struct {
		name  string
		fact  missions.Fact
		moved map[missions.MissionID]uint32
	}{
		{"a take from dz in br", take("fr", "dz", "br"), map[missions.MissionID]uint32{raid: 1}},
		{"a take from dz elsewhere", take("fr", "dz", "es"), map[missions.MissionID]uint32{}},
		{"a box", missions.BoxCaught{At: monday}, map[missions.MissionID]uint32{boxes: 1}},
		{"a right answer", missions.RightAnswer{At: monday}, map[missions.MissionID]uint32{answers: 1}},
		{"a bomb on br", missions.BombLanded{Ground: "br", Position: 1}, map[missions.MissionID]uint32{bombs: 1}},
		{"a bomb in the sea", missions.BombLanded{Position: 1}, map[missions.MissionID]uint32{}},
		{"the shield that fills a tile", missions.ShieldPlaced{Shields: 10, Position: 1}, map[missions.MissionID]uint32{walls: 1}},
		{"a shield short of it", missions.ShieldPlaced{Shields: 9, Position: 1}, map[missions.MissionID]uint32{}},
		{"a fortify of 7 tiles", missions.Fortified{Tiles: 7, Position: 1}, map[missions.MissionID]uint32{fortify: 7}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sheet := missions.SheetOf(missions.SheetSnapshot{}).WithFact(tc.fact, briefOf(t, featBrief...))

			assert.Equal(t, tc.moved, sheet.Snapshot().Progress)
		})
	}
}

func TestAFortifyCountsItsTilesUpToTheGoal(t *testing.T) {
	brief := briefOf(t, featBrief...)

	sheet := missions.SheetOf(missions.SheetSnapshot{}).
		WithFact(missions.Fortified{Tiles: 40, Position: 1}, brief).
		WithFact(missions.Fortified{Tiles: 40, Position: 2}, brief)

	assert.Equal(t, uint32(50), sheet.Snapshot().Progress[fortify])
}

func TestEachKindOfFactKeepsItsOwnMark(t *testing.T) {
	brief := briefOf(t, featBrief...)

	sheet := missions.SheetOf(missions.SheetSnapshot{}).
		WithFact(missions.Take{Flag: "fr", Previous: "dz", Ground: "br", Position: 9}, brief).
		WithFact(missions.BombLanded{Ground: "br", Position: 3}, brief).
		WithFact(missions.ShieldPlaced{Shields: 10, Position: 4}, brief).
		WithFact(missions.Fortified{Tiles: 1, Position: 5}, brief).
		WithFact(missions.BoxCaught{At: monday}, brief).
		WithFact(missions.RightAnswer{At: monday.Add(-time.Hour)}, brief)

	assert.Equal(t, missions.Counted{
		TakesBelow: missions.LedgerMark{Position: 9, Index: 1}, BombsBelow: 4, ShieldsBelow: 5, FortifiesBelow: 6,
		BoxesUntil: monday, AnswersUntil: monday.Add(-time.Hour),
	}, sheet.Snapshot().Counted, "a bomb below the last take still counts")
}

func TestABoxOrAnAnswerToldAgainCountsOnce(t *testing.T) {
	brief := briefOf(t, featBrief...)
	sheet := missions.SheetOf(missions.SheetSnapshot{})

	for range 2 {
		sheet = sheet.WithFact(missions.BoxCaught{At: monday.Add(time.Nanosecond)}, brief).WithFact(missions.RightAnswer{At: monday}, brief)
	}
	sheet = sheet.WithFact(missions.BoxCaught{At: monday.Add(-time.Second)}, brief)

	assert.Equal(t, map[missions.MissionID]uint32{boxes: 1, answers: 1}, sheet.Snapshot().Progress,
		"a time at or before the mark was counted, to the microsecond postgres keeps")
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
	sheet := missions.SheetOf(missions.SheetSnapshot{Progress: map[missions.MissionID]uint32{bank: 2, rival: 5, ground: 4}})

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
	sheet := missions.SheetOf(missions.SheetSnapshot{Progress: map[missions.MissionID]uint32{bank: 2, rival: 3}, Paid: cpcolls.NewSet(bank)})

	assert.Equal(t, []missions.MissionStatus{
		{Mission: frBrief[0], Progress: 2, Done: true, Paid: true},
		{Mission: frBrief[1], Progress: 3},
		{Mission: frBrief[2]},
	}, sheet.Statuses(brief))
}
