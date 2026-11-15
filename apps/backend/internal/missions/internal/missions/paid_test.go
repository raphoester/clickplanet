package missions_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

func paidBrief(t *testing.T, orders ...missions.Order) (missions.Brief, []missions.MissionID) {
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

func paidSheet(ids ...missions.MissionID) missions.Sheet {
	return missions.SheetOf(missions.SheetSnapshot{Paid: cpcolls.NewSet(ids...)})
}

func TestNothingPaidHasNoKind(t *testing.T) {
	assert.Equal(t, missions.Paid{ByKind: map[missions.Kind]missions.KindPaid{}}, missions.NothingPaid())
}

func TestADayAddsItsMissionsPaidByKindWithTheirGoals(t *testing.T) {
	brief, ids := paidBrief(t,
		missions.TakeFor{Flag: "fr", Goal: 120}, missions.TakeFor{Flag: "fr", Goal: 200},
		missions.CatchBoxes{Goal: 3}, missions.FortifyTiles{Goal: 40},
	)

	paid := missions.NothingPaid().WithDay(brief, paidSheet(ids[0], ids[1], ids[2]))

	assert.Equal(t, missions.Paid{InAll: 3, ByKind: map[missions.Kind]missions.KindPaid{
		missions.KindTakeFor:    {Done: 2, Goals: 320},
		missions.KindCatchBoxes: {Done: 1, Goals: 3},
	}}, paid, "a mission done and not paid counts nothing")
}

func TestTheDaysAddUp(t *testing.T) {
	monday, mondays := paidBrief(t, missions.TakeFor{Flag: "fr", Goal: 120}, missions.BombIn{Ground: "br", Goal: 1})
	tuesday, tuesdays := paidBrief(t,
		missions.TakeFor{Flag: "fr", Goal: 180}, missions.TakeFrom{Flag: "dz", Goal: 80}, missions.BombIn{Ground: "is", Goal: 2},
	)
	before := missions.NothingPaid().WithDay(monday, paidSheet(mondays...))

	paid := before.WithDay(tuesday, paidSheet(tuesdays...))

	assert.Equal(t, missions.Paid{InAll: 5, ByKind: map[missions.Kind]missions.KindPaid{
		missions.KindTakeFor:  {Done: 2, Goals: 300},
		missions.KindTakeFrom: {Done: 1, Goals: 80},
		missions.KindBombIn:   {Done: 2, Goals: 3},
	}}, paid)
	assert.Equal(t, uint64(2), before.InAll, "the totals before are left as they were")
	assert.Equal(t, missions.KindPaid{Done: 1, Goals: 120}, before.ByKind[missions.KindTakeFor])
}

func TestAPaymentTheBriefDoesNotHoldCountsInAllOnly(t *testing.T) {
	brief, ids := paidBrief(t, missions.AnswerQuizzes{Goal: 2})

	paid := missions.Paid{}.WithDay(brief, paidSheet(ids[0], missions.MissionID{15: 99}))

	assert.Equal(t, missions.Paid{InAll: 2, ByKind: map[missions.Kind]missions.KindPaid{
		missions.KindAnswerQuizzes: {Done: 1, Goals: 2},
	}}, paid)
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
	brief, ids := paidBrief(t, orders...)

	paid := missions.NothingPaid().WithDay(brief, paidSheet(ids...))

	require.Len(t, paid.ByKind, len(orders))
	assert.ElementsMatch(t, missions.TestingKinds(), keysOf(paid.ByKind))
	var goals uint64
	for _, total := range paid.ByKind {
		assert.Equal(t, uint64(1), total.Done)
		goals += total.Goals
	}
	assert.Equal(t, uint64(45), goals, "each with its own goal")
}

func keysOf(byKind map[missions.Kind]missions.KindPaid) []missions.Kind {
	kinds := make([]missions.Kind, 0, len(byKind))
	for kind := range byKind {
		kinds = append(kinds, kind)
	}
	return kinds
}
