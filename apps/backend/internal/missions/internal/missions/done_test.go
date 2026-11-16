package missions_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func doneBrief(t *testing.T, orders ...missions.Order) (missions.Brief, []missions.MissionID) {
	t.Helper()

	snapshots := make([]missions.MissionSnapshot, 0, len(orders))
	ids := make([]missions.MissionID, 0, len(orders))
	for n, order := range orders {
		id := missions.MissionID{14: byte(len(orders)), 15: byte(n)}
		snapshots = append(snapshots, missions.MissionSnapshot{ID: id, Order: order})
		ids = append(ids, id)
	}
	brief, err := missions.BriefOf(snapshots)
	require.NoError(t, err)
	return brief, ids
}

func progressed(progress map[missions.MissionID]uint32, paid ...missions.MissionID) missions.Sheet {
	return missions.SheetOf(missions.SheetSnapshot{Progress: progress, Paid: cpcolls.NewSet(paid...)})
}

func TestNothingDoneHasNoKind(t *testing.T) {
	assert.Equal(t, missions.Done{ByKind: map[missions.Kind]missions.KindDone{}}, missions.NothingDone())
}

func TestADayAddsTheMissionsWhoseProgressReachedTheirGoalByKindWithTheirGoals(t *testing.T) {
	brief, ids := doneBrief(t,
		missions.TakeFor{Flag: "fr", Goal: 120}, missions.TakeFor{Flag: "fr", Goal: 200},
		missions.CatchBoxes{Goal: 3}, missions.FortifyTiles{Goal: 40},
	)

	done := missions.NothingDone().WithDay(brief, progressed(map[missions.MissionID]uint32{
		ids[0]: 120, ids[1]: 200, ids[2]: 3, ids[3]: 39,
	}))

	assert.Equal(t, missions.Done{InAll: 3, ByKind: map[missions.Kind]missions.KindDone{
		missions.KindTakeFor:    {Done: 2, Goals: 320},
		missions.KindCatchBoxes: {Done: 1, Goals: 3},
	}}, done, "a mission one short of its goal is not done")
}

func TestAMissionDoneCountsWhetherOrNotItWasPaid(t *testing.T) {
	brief, ids := doneBrief(t, missions.AnswerQuizzes{Goal: 2}, missions.BombIn{Ground: "br", Goal: 1})
	progress := map[missions.MissionID]uint32{ids[0]: 2, ids[1]: 1}

	unpaid := missions.NothingDone().WithDay(brief, progressed(progress))
	paid := missions.NothingDone().WithDay(brief, progressed(progress, ids[0]))

	assert.Equal(t, unpaid, paid, "a guest's missions done count as a signed-in player's")
	assert.Equal(t, uint64(2), unpaid.InAll)
}

func TestAPaymentWithNoProgressIsNotAMissionDone(t *testing.T) {
	brief, ids := doneBrief(t, missions.CatchBoxes{Goal: 3})

	assert.Equal(t, missions.NothingDone(), missions.NothingDone().WithDay(brief, progressed(nil, ids[0])))
}

func TestTheDaysAddUp(t *testing.T) {
	monday, mondays := doneBrief(t, missions.TakeFor{Flag: "fr", Goal: 120}, missions.BombIn{Ground: "br", Goal: 1})
	tuesday, tuesdays := doneBrief(t,
		missions.TakeFor{Flag: "fr", Goal: 180}, missions.TakeFrom{Flag: "dz", Goal: 80}, missions.BombIn{Ground: "is", Goal: 2},
	)
	before := missions.NothingDone().WithDay(monday, progressed(map[missions.MissionID]uint32{mondays[0]: 120, mondays[1]: 1}))

	done := before.WithDay(tuesday, progressed(map[missions.MissionID]uint32{tuesdays[0]: 180, tuesdays[1]: 80, tuesdays[2]: 2}))

	assert.Equal(t, missions.Done{InAll: 5, ByKind: map[missions.Kind]missions.KindDone{
		missions.KindTakeFor:  {Done: 2, Goals: 300},
		missions.KindTakeFrom: {Done: 1, Goals: 80},
		missions.KindBombIn:   {Done: 2, Goals: 3},
	}}, done)
	assert.Equal(t, uint64(2), before.InAll, "the totals before are left as they were")
	assert.Equal(t, missions.KindDone{Done: 1, Goals: 120}, before.ByKind[missions.KindTakeFor])
}

func TestEveryKindOfMissionIsCountedUnderAKindOfItsOwn(t *testing.T) {
	orders := []missions.Order{
		missions.TakeFor{Flag: "fr", Goal: 1},
		missions.TakeFrom{Flag: "dz", Goal: 2},
		missions.TakeIn{Ground: "br", Goal: 3},
		missions.TakeFromIn{Flag: "dz", Ground: "br", Goal: 4},
		missions.CatchBoxes{Goal: 5},
		missions.AnswerQuizzes{Goal: 6},
		missions.BombIn{Ground: "br", Goal: 7},
		missions.ShieldTiles{Shields: 10, Goal: 8},
		missions.FortifyTiles{Goal: 9},
	}
	brief, ids := doneBrief(t, orders...)
	progress := map[missions.MissionID]uint32{}
	for n, id := range ids {
		progress[id] = uint32(n + 1) //nolint:gosec // nine missions.
	}

	done := missions.NothingDone().WithDay(brief, progressed(progress))

	require.Len(t, done.ByKind, len(orders))
	kinds := make([]missions.Kind, 0, len(done.ByKind))
	var goals uint64
	for kind, total := range done.ByKind {
		kinds = append(kinds, kind)
		assert.Equal(t, uint64(1), total.Done)
		goals += total.Goals
	}
	assert.ElementsMatch(t, missions.TestingKinds(), kinds)
	assert.Equal(t, uint64(45), goals, "each with its own goal")
}
