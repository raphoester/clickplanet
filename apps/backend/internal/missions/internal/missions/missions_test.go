package missions_test

import (
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
	Territories: []missions.Holding{{Country: "dz", Tiles: 90_000}, {Country: "pw", Tiles: 70_000}, {Country: "fr", Tiles: 300}},
	Grounds: []missions.Holding{
		{Country: "br", Tiles: 9_000}, {Country: "fr", Tiles: 5_000}, {Country: "is", Tiles: 1_200}, {Country: "va", Tiles: 1},
	},
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

func TestTheFirstMissionIsOneBankForTheFlag(t *testing.T) {
	first := missions.Draw(calendar.DayOf(monday), "fr", world).Snapshot()[0]

	assert.Equal(t, missions.MissionSnapshot{
		ID: first.ID, Kind: missions.KindTakeFor, Target: "fr", Goal: 60,
		Reward: missions.Reward{Kind: missions.RewardShields, Amount: 10},
	}, first)
}

func TestTheSecondMissionAimsAtTheLeaderOrTheRunnerUpForTheLeader(t *testing.T) {
	day := calendar.DayOf(monday)

	second := missions.Draw(day, "fr", world).Snapshot()[1]

	assert.Equal(t, missions.MissionSnapshot{
		ID: second.ID, Kind: missions.KindTakeFrom, Target: "dz", Goal: 40,
		Reward: missions.Reward{Kind: missions.RewardSpreadClicks, Amount: 4},
	}, second)
	assert.Equal(t, "pw", missions.Draw(day, "dz", world).Snapshot()[1].Target)
}

func TestWithNoRivalBigEnoughTheSecondMissionIsTwoBanks(t *testing.T) {
	empty := missions.World{Territories: []missions.Holding{{Country: "de", Tiles: 399}}}

	second := missions.Draw(calendar.DayOf(monday), "fr", empty).Snapshot()[1]

	assert.Equal(t, missions.KindTakeFor, second.Kind)
	assert.Equal(t, "fr", second.Target)
	assert.Equal(t, uint32(120), second.Goal)
}

func TestTheThirdMissionIsABigGroundThatIsNotTheFlags(t *testing.T) {
	for _, flag := range []string{"fr", "br", "is", "dz"} {
		for days := range 30 {
			third := missions.Draw(calendar.DayOf(monday.AddDate(0, 0, days)), flag, world).Snapshot()[2]

			assert.Equal(t, missions.KindTakeIn, third.Kind)
			assert.Contains(t, []string{"br", "fr", "is"}, third.Target, "no ground under 1,000 tiles")
			assert.NotEqual(t, flag, third.Target)
			assert.Equal(t, missions.Reward{Kind: missions.RewardBomb, Amount: 1}, third.Reward)
		}
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

func TestAKindNoMissionHasIsRefused(t *testing.T) {
	_, err := missions.BriefOf([]missions.MissionSnapshot{{ID: bank, Kind: "fortify", Target: "is", Goal: 1}})

	assert.ErrorIs(t, err, missions.ErrUnknownKind)
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
	{ID: bank, Kind: missions.KindTakeFor, Target: "fr", Goal: 2},
	{ID: rival, Kind: missions.KindTakeFrom, Target: "dz", Goal: 5},
	{ID: ground, Kind: missions.KindTakeIn, Target: "br", Goal: 5},
}

func TestTheFirstTakeOfTheDayEnlistsItsFlag(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{})
	require.False(t, sheet.Enlisted())

	first := take("fr", "", "es")
	sheet = sheet.WithTake(first, briefOf(t, frBrief...))

	assert.True(t, sheet.Enlisted())
	assert.Equal(t, missions.SheetSnapshot{
		Flag: "fr", Taken: map[missions.MissionID]uint32{bank: 1}, Paid: cpcolls.NewSet[missions.MissionID](),
		TakesBelow: first.Position + 1,
	}, sheet.Snapshot())
}

func TestATakeCountsForEveryMissionItFits(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{}).WithTake(take("fr", "dz", "br"), briefOf(t, frBrief...))

	assert.Equal(t, map[missions.MissionID]uint32{bank: 1, rival: 1, ground: 1}, sheet.Snapshot().Taken)
}

func TestATakeForAnotherFlagThanTheDaysCountsNothing(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{Flag: "fr"})

	sheet = sheet.WithTake(take("de", "dz", "br"), briefOf(
		t,
		missions.MissionSnapshot{ID: bank, Kind: missions.KindTakeFor, Target: "de", Goal: 60},
		missions.MissionSnapshot{ID: rival, Kind: missions.KindTakeFrom, Target: "dz", Goal: 60},
	))

	assert.Equal(t, "fr", sheet.Snapshot().Flag, "the day's flag is kept")
	assert.Empty(t, sheet.Snapshot().Taken)
}

func TestAMissionCountsNoFurtherThanItsGoal(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{})
	for range 5 {
		sheet = sheet.WithTake(take("fr", "", ""), briefOf(t, frBrief...))
	}

	assert.Equal(t, uint32(2), sheet.Snapshot().Taken[bank])
}

func TestATakeToldTwiceCountsOnce(t *testing.T) {
	brief := briefOf(t, frBrief...)
	told := take("fr", "dz", "")

	sheet := missions.SheetOf(missions.SheetSnapshot{}).WithTake(told, brief).WithTake(told, brief)

	assert.Equal(t, map[missions.MissionID]uint32{bank: 1, rival: 1}, sheet.Snapshot().Taken)
	assert.Equal(t, told.Position+1, sheet.Snapshot().TakesBelow)
}

func TestATakeBelowTheMarkCountsNothing(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{Flag: "fr", TakesBelow: 1_000})

	sheet = sheet.WithTake(missions.Take{Account: ada, Flag: "fr", At: monday, Position: 999}, briefOf(t, frBrief...))

	assert.Empty(t, sheet.Snapshot().Taken)
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
	assert.Empty(t, before.Snapshot().Taken)
}

func TestEachMissionDoneIsPaidOnce(t *testing.T) {
	shields := missions.Reward{Kind: missions.RewardShields, Amount: 10}
	bomb := missions.Reward{Kind: missions.RewardBomb, Amount: 1}
	brief := briefOf(t,
		missions.MissionSnapshot{ID: bank, Kind: missions.KindTakeFor, Target: "fr", Goal: 2, Reward: shields},
		missions.MissionSnapshot{ID: rival, Kind: missions.KindTakeFrom, Target: "dz", Goal: 5, Reward: bomb},
		missions.MissionSnapshot{ID: ground, Kind: missions.KindTakeIn, Target: "br", Goal: 5, Reward: bomb},
	)
	sheet := missions.SheetOf(missions.SheetSnapshot{Flag: "fr", Taken: map[missions.MissionID]uint32{bank: 2, rival: 5, ground: 4}})

	paid, payments := sheet.WithPayments(brief)
	_, again := paid.WithPayments(brief)

	assert.Equal(t, []missions.Payment{{Mission: bank, Reward: shields}, {Mission: rival, Reward: bomb}}, payments,
		"4 of 5 is not done")
	assert.Empty(t, again)
	assert.Equal(t, cpcolls.NewSet(bank, rival), paid.Snapshot().Paid)
	assert.True(t, sheet.Snapshot().Paid.Empty(), "the sheet it was called on is left alone")
}
