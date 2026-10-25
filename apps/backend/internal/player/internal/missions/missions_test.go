package missions_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var (
	calendar = missions.NewCalendar(missions.Config{Cutoff: 21 * time.Hour})
	monday   = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	ada      = players.AccountID{15: 1}
)

var world = missions.World{
	Territories: []missions.Holding{{Country: "dz", Tiles: 90_000}, {Country: "pw", Tiles: 70_000}, {Country: "fr", Tiles: 300}},
	Grounds: []missions.Holding{
		{Country: "br", Tiles: 9_000}, {Country: "fr", Tiles: 5_000}, {Country: "is", Tiles: 1_200}, {Country: "va", Tiles: 1},
	},
}

func take(flag, previous, ground string) missions.Take {
	return missions.Take{Account: ada, Flag: flag, Previous: previous, Ground: ground, At: monday}
}

func snapshotsOf(brief missions.Brief) map[missions.Slot]missions.MissionSnapshot {
	snapshots := map[missions.Slot]missions.MissionSnapshot{}
	for _, snapshot := range brief.Snapshot() {
		snapshots[snapshot.Slot] = snapshot
	}
	return snapshots
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
	first := snapshotsOf(missions.Draw(calendar.DayOf(monday), "fr", world))[1]

	assert.Equal(t, missions.MissionSnapshot{
		Slot: 1, Kind: missions.KindTakeFor, Target: "fr", Goal: 60,
		Reward: missions.Reward{Kind: missions.RewardShields, Amount: 10},
	}, first)
}

func TestTheSecondMissionAimsAtTheLeaderOrTheRunnerUpForTheLeader(t *testing.T) {
	day := calendar.DayOf(monday)

	assert.Equal(t, missions.MissionSnapshot{
		Slot: 2, Kind: missions.KindTakeFrom, Target: "dz", Goal: 40,
		Reward: missions.Reward{Kind: missions.RewardSpreadClicks, Amount: 4},
	}, snapshotsOf(missions.Draw(day, "fr", world))[2])
	assert.Equal(t, "pw", snapshotsOf(missions.Draw(day, "dz", world))[2].Target)
}

func TestWithNoRivalBigEnoughTheSecondMissionIsTwoBanks(t *testing.T) {
	empty := missions.World{Territories: []missions.Holding{{Country: "de", Tiles: 399}}}

	second := snapshotsOf(missions.Draw(calendar.DayOf(monday), "fr", empty))[2]

	assert.Equal(t, missions.KindTakeFor, second.Kind)
	assert.Equal(t, "fr", second.Target)
	assert.Equal(t, uint32(120), second.Goal)
}

func TestTheThirdMissionIsABigGroundThatIsNotTheFlags(t *testing.T) {
	for _, flag := range []string{"fr", "br", "is", "dz"} {
		for days := range 30 {
			third := snapshotsOf(missions.Draw(calendar.DayOf(monday.AddDate(0, 0, days)), flag, world))[3]

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

func TestABriefReadsBackFromItsSnapshot(t *testing.T) {
	drawn := missions.Draw(calendar.DayOf(monday), "fr", world)

	restored, err := missions.BriefOf(drawn.Snapshot())

	require.NoError(t, err)
	assert.Equal(t, drawn.Snapshot(), restored.Snapshot())
}

func TestAKindNoMissionHasIsRefused(t *testing.T) {
	_, err := missions.BriefOf([]missions.MissionSnapshot{{Slot: 1, Kind: "fortify", Target: "is", Goal: 1}})

	assert.ErrorIs(t, err, missions.ErrUnknownKind)
}

func TestOnlyThreeSlotsExist(t *testing.T) {
	slot, err := missions.SlotOf(3)
	require.NoError(t, err)
	assert.Equal(t, missions.Slot(3), slot)

	_, err = missions.SlotOf(4)
	assert.ErrorIs(t, err, missions.ErrNoSlot)
}

func briefOf(t *testing.T, snapshots ...missions.MissionSnapshot) missions.Brief {
	t.Helper()

	brief, err := missions.BriefOf(snapshots)
	require.NoError(t, err)
	return brief
}

var frBrief = []missions.MissionSnapshot{
	{Slot: 1, Kind: missions.KindTakeFor, Target: "fr", Goal: 2},
	{Slot: 2, Kind: missions.KindTakeFrom, Target: "dz", Goal: 5},
	{Slot: 3, Kind: missions.KindTakeIn, Target: "br", Goal: 5},
}

func TestTheFirstTakeOfTheDayEnlistsItsFlag(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{})
	require.False(t, sheet.Enlisted())

	sheet = sheet.WithTake(take("fr", "", "es"), briefOf(t, frBrief...))

	assert.True(t, sheet.Enlisted())
	assert.Equal(t, missions.SheetSnapshot{
		Flag: "fr", Taken: map[missions.Slot]uint32{1: 1}, Collected: cpcolls.NewSet[missions.Slot](),
	}, sheet.Snapshot())
}

func TestATakeCountsForEveryMissionItFits(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{}).WithTake(take("fr", "dz", "br"), briefOf(t, frBrief...))

	assert.Equal(t, map[missions.Slot]uint32{1: 1, 2: 1, 3: 1}, sheet.Snapshot().Taken)
}

func TestATakeForAnotherFlagThanTheDaysCountsNothing(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{Flag: "fr"})

	sheet = sheet.WithTake(take("de", "dz", "br"), briefOf(
		t,
		missions.MissionSnapshot{Slot: 1, Kind: missions.KindTakeFor, Target: "de", Goal: 60},
		missions.MissionSnapshot{Slot: 2, Kind: missions.KindTakeFrom, Target: "dz", Goal: 60},
	))

	assert.Equal(t, "fr", sheet.Snapshot().Flag, "the day's flag is kept")
	assert.Empty(t, sheet.Snapshot().Taken)
}

func TestAMissionCountsNoFurtherThanItsGoal(t *testing.T) {
	sheet := missions.SheetOf(missions.SheetSnapshot{})
	for range 5 {
		sheet = sheet.WithTake(take("fr", "", ""), briefOf(t, frBrief...))
	}

	assert.Equal(t, uint32(2), sheet.Snapshot().Taken[1])
}

func TestAWithTakeLeavesTheSheetItWasCalledOnAlone(t *testing.T) {
	before := missions.SheetOf(missions.SheetSnapshot{})

	_ = before.WithTake(take("fr", "", ""), briefOf(t, frBrief...))

	assert.False(t, before.Enlisted())
	assert.Empty(t, before.Snapshot().Taken)
}

func TestAMissionIsCollectedOnceAndOnlyWhenDone(t *testing.T) {
	brief := briefOf(t, frBrief...)
	first, err := brief.Mission(1)
	require.NoError(t, err)

	assert.ErrorIs(t, missions.SheetOf(missions.SheetSnapshot{Flag: "fr", Taken: map[missions.Slot]uint32{1: 1}}).
		CollectError(first), missions.ErrNotDone)
	require.NoError(t, missions.SheetOf(missions.SheetSnapshot{Flag: "fr", Taken: map[missions.Slot]uint32{1: 2}}).
		CollectError(first))
	assert.ErrorIs(t, missions.SheetOf(missions.SheetSnapshot{
		Flag: "fr", Taken: map[missions.Slot]uint32{1: 2}, Collected: cpcolls.NewSet[missions.Slot](1),
	}).CollectError(first), missions.ErrCollected)
}
