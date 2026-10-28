package missions_paid_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	missionsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/missions/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/missions_paid_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

func TestTheMissionsPaidAreTheMissionsDoneOfTheAccount(t *testing.T) {
	store := inmemory_player_store.New()
	events := cpredis.NewRecordedEvents()
	handler := missions_paid_subscriber.New(record_stats_usecase.New(store, events))

	for _, inAll := range []uint64{2, 3, 3} {
		require.NoError(t, handler.Handle(t.Context(), &missionsv1.MissionsPaid{AccountId: account, PaidInAll: inAll}))
	}

	id, err := players.AccountIDOf(account)
	require.NoError(t, err)
	stats, err := store.Stats(t.Context(), id)
	require.NoError(t, err)
	assert.Equal(t, uint64(3), stats.MissionsDone(), "a payment told twice counts once")
	assert.Len(t, events.Published(), 3, "each one may earn a title")
}

func TestMissionsPaidWithNoAccountIsRefused(t *testing.T) {
	handler := missions_paid_subscriber.New(record_stats_usecase.New(inmemory_player_store.New(), cpredis.NewRecordedEvents()))

	assert.ErrorIs(t, handler.Handle(t.Context(), &missionsv1.MissionsPaid{PaidInAll: 1}), players.ErrInvalidAccount)
}
