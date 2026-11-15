package payroll_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inmemory_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/paidmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/payroll"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/planetgrant"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

type accounts struct {
	linked bool
	err    error
	asked  int
}

func (a *accounts) Linked(context.Context, missions.AccountID) (bool, error) {
	a.asked++
	return a.linked, a.err
}

var (
	day      = days.TestingDayAt(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC))
	ada      = missions.AccountID{15: 1}
	drawn    = missions.Draw(day, ada, "fr", missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}})
	bank     = drawn.Snapshot()[0]
	bankGoal = bank.Order.(missions.TakeFor).Goal
)

func setUp(t *testing.T, accounts *accounts, taken uint32) (payroll.Payroll, *inmemory_mission_store.Store, *cpatomic.FakeOutbox) {
	t.Helper()

	store := inmemory_mission_store.New()
	_, err := store.KeepBrief(t.Context(), day, ada, drawn)
	require.NoError(t, err)
	require.NoError(t, store.ChangeSheet(t.Context(), day, ada, func(missions.Sheet) missions.Sheet {
		return missions.SheetOf(missions.SheetSnapshot{Progress: map[missions.MissionID]uint32{bank.ID: taken}})
	}))
	outbox := &cpatomic.FakeOutbox{}
	return payroll.New(store, accounts, outbox), store, outbox
}

func pay(t *testing.T, paying payroll.Payroll) error {
	t.Helper()

	return cpatomic.FakeTransactioner{}.InTx(t.Context(), func(ctx context.Context) error {
		return paying.Pay(ctx, day, ada)
	})
}

func TestASignedInAccountIsGrantedEachMissionDoneOnce(t *testing.T) {
	paying, store, outbox := setUp(t, &accounts{linked: true}, bankGoal)

	require.NoError(t, pay(t, paying))
	require.NoError(t, pay(t, paying))

	require.Len(t, outbox.Recorded(), 2)
	assert.True(t, proto.Equal(planetgrant.Of(ada, missions.Payment{Mission: bank.ID, Reward: bank.Reward}), outbox.Recorded()[0]))
	assert.NotEqual(t, planetv1.BonusKind_BONUS_KIND_UNSPECIFIED, outbox.Recorded()[0].(*planetv1.GrantCharges).GetKind())
	assert.True(t, proto.Equal(&missionsv1.MissionsPaid{AccountId: ada.String(), PaidInAll: 1, ByKind: []*missionsv1.KindPaid{
		{Kind: missionsv1.MissionKind_MISSION_KIND_TAKE_FOR, Done: 1, Goals: uint64(bankGoal)},
	}}, outbox.Recorded()[1]), outbox.Recorded()[1])
	sheet, err := store.Sheet(t.Context(), day, ada)
	require.NoError(t, err)
	assert.True(t, sheet.Snapshot().Paid.Contains(bank.ID))
}

func TestTheMissionsPaidCountEveryDayOfTheAccount(t *testing.T) {
	paying, store, outbox := setUp(t, &accounts{linked: true}, bankGoal)
	yesterday := days.TestingDayAt(time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC))
	require.NoError(t, store.ChangeSheet(t.Context(), yesterday, ada, func(missions.Sheet) missions.Sheet {
		return missions.SheetOf(missions.SheetSnapshot{Paid: cpcolls.NewSet(missions.MissionID{15: 1}, missions.MissionID{15: 2})})
	}))

	require.NoError(t, pay(t, paying))

	told := outbox.Recorded()[1].(*missionsv1.MissionsPaid)
	assert.Equal(t, uint64(3), told.GetPaidInAll())
	assert.Len(t, told.GetByKind(), 1, "yesterday's brief was never kept: its payments have no kind")
}

func TestTheMissionsPaidSumTheGoalsOfEachKindOverEveryDay(t *testing.T) {
	paying, store, outbox := setUp(t, &accounts{linked: true}, bankGoal)
	yesterday := days.TestingDayAt(time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC))
	before := missions.Draw(yesterday, ada, "fr", missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}})
	_, err := store.KeepBrief(t.Context(), yesterday, ada, before)
	require.NoError(t, err)
	paidBefore := before.Snapshot()[:2]
	require.NoError(t, store.ChangeSheet(t.Context(), yesterday, ada, func(missions.Sheet) missions.Sheet {
		return missions.SheetOf(missions.SheetSnapshot{Paid: cpcolls.NewSet(paidBefore[0].ID, paidBefore[1].ID)})
	}))

	require.NoError(t, pay(t, paying))

	expected := missions.NothingPaid().
		WithDay(before, missions.SheetOf(missions.SheetSnapshot{Paid: cpcolls.NewSet(paidBefore[0].ID, paidBefore[1].ID)})).
		WithDay(drawn, missions.SheetOf(missions.SheetSnapshot{Paid: cpcolls.NewSet(bank.ID)}))
	assert.True(t, proto.Equal(paidmessage.Of(ada, expected), outbox.Recorded()[1]), outbox.Recorded()[1])
	assert.Equal(t, uint64(bankGoal+paidBefore[0].Order.(missions.TakeFor).Goal),
		outbox.Recorded()[1].(*missionsv1.MissionsPaid).GetByKind()[0].GetGoals(), "both banks' goals")
}

func TestAGuestIsNotPaidYet(t *testing.T) {
	paying, store, outbox := setUp(t, &accounts{linked: false}, bankGoal)

	require.NoError(t, pay(t, paying))

	assert.Empty(t, outbox.Recorded())
	sheet, err := store.Sheet(t.Context(), day, ada)
	require.NoError(t, err)
	assert.True(t, sheet.Snapshot().Paid.Empty(), "it is paid when it signs in")
}

func TestWithNothingDoneAuthIsNotAsked(t *testing.T) {
	asked := &accounts{linked: true}
	paying, _, outbox := setUp(t, asked, bankGoal-1)

	require.NoError(t, pay(t, paying))

	assert.Zero(t, asked.asked)
	assert.Empty(t, outbox.Recorded())
}

func TestADayNeverDrawnPaysNothing(t *testing.T) {
	asked := &accounts{linked: true}
	outbox := &cpatomic.FakeOutbox{}

	require.NoError(t, pay(t, payroll.New(inmemory_mission_store.New(), asked, outbox)))

	assert.Zero(t, asked.asked)
	assert.Empty(t, outbox.Recorded())
}

func TestAnAuthThatFailsPaysNothing(t *testing.T) {
	paying, _, outbox := setUp(t, &accounts{err: errors.New("down")}, bankGoal)

	require.Error(t, pay(t, paying))

	assert.Empty(t, outbox.Recorded())
}
