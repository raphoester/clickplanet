package retell_missions_paid_usecase_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inmemory_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/paidmessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/retell_missions_paid_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
)

var (
	day   = days.TestingDayAt(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC))
	world = missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}
)

func account(i int) missions.AccountID {
	return missions.AccountID{14: byte(i >> 8), 15: byte(i)}
}

// The account's first paid missions of its brief of the day.
func paid(t *testing.T, store *inmemory_mission_store.Store, i, missionsPaid int) {
	t.Helper()

	brief, err := store.KeepBrief(t.Context(), day, account(i), missions.Draw(day, account(i), "fr", world))
	require.NoError(t, err)
	ids := make([]missions.MissionID, 0, missionsPaid)
	for _, mission := range brief.Snapshot()[:missionsPaid] {
		ids = append(ids, mission.ID)
	}
	require.NoError(t, store.ChangeSheet(t.Context(), day, account(i), func(missions.Sheet) missions.Sheet {
		return missions.SheetOf(missions.SheetSnapshot{Paid: cpcolls.NewSet(ids...)})
	}))
}

func retell(t *testing.T, store *inmemory_mission_store.Store, outbox cpatomic.Outbox, dryRun bool) (retell_missions_paid_usecase.Out, error) {
	t.Helper()

	return retell_missions_paid_usecase.New(store, cpatomic.FakeTransactioner{}, outbox). //nolint:wrapcheck // the tests read the error.
												Execute(t.Context(), retell_missions_paid_usecase.In{DryRun: dryRun})
}

func told(outbox *cpatomic.FakeOutbox) []*missionsv1.MissionsPaid {
	recorded := outbox.Recorded()
	events := make([]*missionsv1.MissionsPaid, 0, len(recorded))
	for _, event := range recorded {
		events = append(events, event.(*missionsv1.MissionsPaid))
	}
	return events
}

func TestEveryAccountPaidIsToldWhatItWasPaidInAllByKind(t *testing.T) {
	store, outbox := inmemory_mission_store.New(), &cpatomic.FakeOutbox{}
	paid(t, store, 1, 2)
	paid(t, store, 2, 1)
	paid(t, store, 3, 0)

	out, err := retell(t, store, outbox, false)

	require.NoError(t, err)
	assert.Equal(t, 2, out.Accounts, "an account with no mission paid is not told")
	require.Len(t, told(outbox), 2)
	for n, i := range []int{1, 2} {
		totals, err := store.PaidTotals(t.Context(), account(i))
		require.NoError(t, err)
		assert.True(t, proto.Equal(paidmessage.Of(account(i), totals), told(outbox)[n]), told(outbox)[n])
		assert.NotEmpty(t, told(outbox)[n].GetByKind())
	}
	assert.Equal(t, uint64(2), told(outbox)[0].GetPaidInAll())
}

func TestADryRunCountsTheAccountsAndTellsNothing(t *testing.T) {
	store, outbox := inmemory_mission_store.New(), &cpatomic.FakeOutbox{}
	paid(t, store, 1, 1)
	paid(t, store, 2, 3)

	out, err := retell(t, store, outbox, true)

	require.NoError(t, err)
	assert.Equal(t, 2, out.Accounts)
	assert.Empty(t, outbox.Recorded())
}

func TestASecondRunTellsTheSameTotals(t *testing.T) {
	store, outbox := inmemory_mission_store.New(), &cpatomic.FakeOutbox{}
	paid(t, store, 1, 2)

	_, err := retell(t, store, outbox, false)
	require.NoError(t, err)
	_, err = retell(t, store, outbox, false)
	require.NoError(t, err)

	require.Len(t, told(outbox), 2)
	assert.True(t, proto.Equal(told(outbox)[0], told(outbox)[1]), "a listener that keeps the highest totals changes nothing")
}

func TestEveryPageIsToldEachAccountOnce(t *testing.T) {
	store, outbox := inmemory_mission_store.New(), &cpatomic.FakeOutbox{}
	const accounts = 1_001
	for i := range accounts {
		paid(t, store, i+1, 1)
	}

	out, err := retell(t, store, outbox, false)

	require.NoError(t, err)
	assert.Equal(t, accounts, out.Accounts)
	ids := make([]string, 0, accounts)
	for _, event := range told(outbox) {
		ids = append(ids, event.GetAccountId())
	}
	assert.True(t, slices.IsSorted(ids), "in the order of the accounts, so no page tells one twice")
	assert.Len(t, slices.Compact(ids), accounts)
}

func TestAStoreThatFailsTellsNothing(t *testing.T) {
	store, outbox := inmemory_mission_store.New(), &cpatomic.FakeOutbox{}
	paid(t, store, 1, 1)
	store.FailWith(errors.New("down"))

	_, err := retell(t, store, outbox, false)

	require.Error(t, err)
	assert.Empty(t, outbox.Recorded())
}

type failingOutbox struct{}

func (failingOutbox) Record(context.Context, ...proto.Message) error { return errors.New("full") }

func TestAnOutboxThatFailsIsAnError(t *testing.T) {
	store := inmemory_mission_store.New()
	paid(t, store, 1, 1)

	_, err := retell(t, store, failingOutbox{}, false)

	require.ErrorContains(t, err, "failed to record a page of the missions paid")
}
