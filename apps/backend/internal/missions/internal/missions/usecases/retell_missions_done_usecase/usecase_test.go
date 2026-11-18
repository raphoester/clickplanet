package retell_missions_done_usecase_test

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/donemessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/inmemory_mission_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/retell_missions_done_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var (
	day   = days.TestingDayAt(time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC))
	world = missions.World{Territories: []missions.Territory{{Flag: "dz", Tiles: 9_000}}}
)

func account(i int) missions.AccountID {
	return missions.AccountID{14: byte(i >> 8), 15: byte(i)}
}

// The account's first missions of its brief of the day taken to their goals, paid or not.
func done(t *testing.T, store *inmemory_mission_store.Store, i, missionsDone int) {
	t.Helper()

	brief, err := store.KeepBrief(t.Context(), day, account(i), missions.Draw(day, account(i), "fr", world))
	require.NoError(t, err)
	progress := map[missions.MissionID]uint32{}
	for _, mission := range brief.Snapshot()[:missionsDone] {
		progress[mission.ID] = 1_000
	}
	require.NoError(t, store.ChangeSheet(t.Context(), day, account(i), func(missions.Sheet) missions.Sheet {
		return missions.SheetOf(missions.SheetSnapshot{Progress: progress})
	}))
}

func retell(t *testing.T, store *inmemory_mission_store.Store, outbox cpatomic.Outbox, dryRun bool) (retell_missions_done_usecase.Out, error) {
	t.Helper()

	return retell_missions_done_usecase.New(store, cpatomic.FakeTransactioner{}, outbox). //nolint:wrapcheck // the tests read the error.
												Execute(t.Context(), retell_missions_done_usecase.In{DryRun: dryRun})
}

func told(outbox *cpatomic.FakeOutbox) []*missionsv1.MissionsDone {
	recorded := outbox.Recorded()
	events := make([]*missionsv1.MissionsDone, 0, len(recorded))
	for _, event := range recorded {
		events = append(events, event.(*missionsv1.MissionsDone))
	}
	return events
}

func TestEveryAccountWithAMissionDoneIsToldWhatItHasDoneInAllByKind(t *testing.T) {
	store, outbox := inmemory_mission_store.New(), &cpatomic.FakeOutbox{}
	done(t, store, 1, 2)
	done(t, store, 2, 1)
	done(t, store, 3, 0)

	out, err := retell(t, store, outbox, false)

	require.NoError(t, err)
	assert.Equal(t, 2, out.Accounts, "an account with no mission done is not told")
	require.Len(t, told(outbox), 2)
	for n, i := range []int{1, 2} {
		totals, err := store.DoneBy(t.Context(), account(i))
		require.NoError(t, err)
		assert.True(t, proto.Equal(donemessage.Of(account(i), totals), told(outbox)[n]), told(outbox)[n])
		assert.NotEmpty(t, told(outbox)[n].GetByKind())
	}
	assert.Equal(t, uint64(2), told(outbox)[0].GetDoneInAll())
}

func TestADryRunCountsTheAccountsAndTellsNothing(t *testing.T) {
	store, outbox := inmemory_mission_store.New(), &cpatomic.FakeOutbox{}
	done(t, store, 1, 1)
	done(t, store, 2, 3)

	out, err := retell(t, store, outbox, true)

	require.NoError(t, err)
	assert.Equal(t, 2, out.Accounts)
	assert.Empty(t, outbox.Recorded())
}

func TestASecondRunTellsTheSameTotals(t *testing.T) {
	store, outbox := inmemory_mission_store.New(), &cpatomic.FakeOutbox{}
	done(t, store, 1, 2)

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
		done(t, store, i+1, 1)
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
