package missions_done_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/missions_done_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

func setUp() (*inmemory_stats_store.Store, *cpredis.RecordedEvents, missions_done_subscriber.Subscriber) {
	store := inmemory_stats_store.New()
	events := cpredis.NewRecordedEvents()
	return store, events, missions_done_subscriber.New(record_stats_usecase.New(store, events))
}

func done(kind missionsv1.MissionKind, count, goals uint64) *missionsv1.KindDone {
	return &missionsv1.KindDone{Kind: kind, Done: count, Goals: goals}
}

func TestEachKindKeepsTheHighestTotalsMissionsTold(t *testing.T) {
	store, events, handler := setUp()

	for _, byKind := range [][]*missionsv1.KindDone{
		{done(missionsv1.MissionKind_MISSION_KIND_TAKE_FOR, 2, 360), done(missionsv1.MissionKind_MISSION_KIND_TAKE_FROM_IN, 1, 80)},
		{done(missionsv1.MissionKind_MISSION_KIND_TAKE_FOR, 1, 180)},
		{done(missionsv1.MissionKind_MISSION_KIND_TAKE_FOR, 2, 360), done(missionsv1.MissionKind_MISSION_KIND_TAKE_FROM_IN, 1, 80)},
	} {
		require.NoError(t, handler.Handle(t.Context(), &missionsv1.MissionsDone{AccountId: account, ByKind: byKind}))
	}

	id, err := accounts.AccountIDOf(account)
	require.NoError(t, err)
	counted, err := store.Stats(t.Context(), id)
	require.NoError(t, err)
	assert.Equal(t, map[stats.MissionKind]stats.MissionTally{
		"take_for": {Done: 2, Goals: 360}, "take_from_in": {Done: 1, Goals: 80},
	}, counted.MissionsByKind(), "named as missions names it in its briefs; a total told late or twice lowers nothing")
	assert.Zero(t, counted.MissionsDone(), "the Missions title track counts what was paid")
	assert.Len(t, events.Published(), 3, "each one may raise the level")
}

func TestAnUnspecifiedKindCountsNothing(t *testing.T) {
	store, _, handler := setUp()

	require.NoError(t, handler.Handle(t.Context(), &missionsv1.MissionsDone{AccountId: account, ByKind: []*missionsv1.KindDone{
		done(missionsv1.MissionKind_MISSION_KIND_UNSPECIFIED, 1, 5), done(missionsv1.MissionKind_MISSION_KIND_BOMB_IN, 1, 2),
	}}))

	id, err := accounts.AccountIDOf(account)
	require.NoError(t, err)
	counted, err := store.Stats(t.Context(), id)
	require.NoError(t, err)
	assert.Equal(t, map[stats.MissionKind]stats.MissionTally{"bomb_in": {Done: 1, Goals: 2}}, counted.MissionsByKind())
}

func TestAKindThisBuildDoesNotKnowIsRefused(t *testing.T) {
	_, events, handler := setUp()

	err := handler.Handle(t.Context(), &missionsv1.MissionsDone{AccountId: account, ByKind: []*missionsv1.KindDone{done(99, 1, 5)}})

	require.ErrorIs(t, err, missions_done_subscriber.ErrUnknownKind)
	assert.Empty(t, events.Published())
}

func TestMissionsDoneWithNoAccountIsRefused(t *testing.T) {
	_, _, handler := setUp()

	assert.ErrorIs(t, handler.Handle(t.Context(), &missionsv1.MissionsDone{DoneInAll: 1}), accounts.ErrInvalidAccount)
}
