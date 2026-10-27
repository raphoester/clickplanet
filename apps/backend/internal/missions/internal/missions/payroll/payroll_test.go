package payroll_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inmemory_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/payroll"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/planetgrant"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type fixedPlanet struct{}

func (fixedPlanet) World(context.Context) (missions.World, error) {
	return missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}, nil
}

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
	day      = missions.NewCalendar(missions.Config{Cutoff: 21 * time.Hour}).DayOf(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC))
	ada      = missions.AccountID{15: 1}
	bank     = missions.Draw(day, "fr", missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}).Snapshot()[0]
	bankGoal = bank.Order.(missions.TakeFor).Goal
)

func setUp(t *testing.T, accounts *accounts, taken uint32) (payroll.Payroll, *inmemory_mission_store.Store, *cpatomic.FakeOutbox) {
	t.Helper()

	store := inmemory_mission_store.New()
	require.NoError(t, store.ChangeSheet(t.Context(), day, ada, func(missions.Sheet) missions.Sheet {
		return missions.SheetOf(missions.SheetSnapshot{Flag: "fr", Progress: map[missions.MissionID]uint32{bank.ID: taken}})
	}))
	outbox := &cpatomic.FakeOutbox{}
	return payroll.New(store, missions.NewBriefing(store, fixedPlanet{}), accounts, outbox), store, outbox
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

	require.Len(t, outbox.Recorded(), 1)
	assert.True(t, proto.Equal(planetgrant.Of(ada, missions.Payment{Mission: bank.ID, Reward: bank.Reward}), outbox.Recorded()[0]))
	assert.NotEqual(t, planetv1.BonusKind_BONUS_KIND_UNSPECIFIED, outbox.Recorded()[0].(*planetv1.GrantCharges).GetKind())
	sheet, err := store.Sheet(t.Context(), day, ada)
	require.NoError(t, err)
	assert.True(t, sheet.Snapshot().Paid.Contains(bank.ID))
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

func TestAnAuthThatFailsPaysNothing(t *testing.T) {
	paying, _, outbox := setUp(t, &accounts{err: errors.New("down")}, bankGoal)

	require.Error(t, pay(t, paying))

	assert.Empty(t, outbox.Recorded())
}
