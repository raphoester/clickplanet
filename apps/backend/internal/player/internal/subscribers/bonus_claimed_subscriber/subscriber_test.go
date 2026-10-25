package bonus_claimed_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/record_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/bonus_claimed_subscriber"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpbootstrap"
)

const account = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"

var at = time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

func subscriber() (bonus_claimed_subscriber.Subscriber, *inmemory_player_store.Store) {
	store := inmemory_player_store.New()
	return bonus_claimed_subscriber.New(record_stats_usecase.New(store, cpbootstrap.NewRecordedEvents())), store
}

func statsOf(t *testing.T, store *inmemory_player_store.Store) players.StatsSnapshot {
	t.Helper()

	id, err := players.AccountIDOf(account)
	require.NoError(t, err)
	stats, err := store.Stats(t.Context(), id)
	require.NoError(t, err)
	return stats.TakeSnapshot()
}

func TestEachClaimCountsOnTheStatsOfTheAccountThatCaughtTheBox(t *testing.T) {
	handler, store := subscriber()

	claimed := func(at time.Time) *planetv1.BonusClaimed {
		return &planetv1.BonusClaimed{AccountId: account, ClaimedAt: timestamppb.New(at)}
	}

	require.NoError(t, handler.Handle(t.Context(), claimed(at)))
	require.NoError(t, handler.Handle(t.Context(), claimed(at.Add(time.Minute))))
	require.NoError(t, handler.Handle(t.Context(), claimed(at.Add(time.Minute))))

	assert.Equal(t, uint64(2), statsOf(t, store).BonusesClaimed, "a claim told twice counts once")
}

func TestAClaimWithNoAccountIsRefused(t *testing.T) {
	handler, _ := subscriber()

	assert.ErrorIs(t, handler.Handle(t.Context(), &planetv1.BonusClaimed{}), players.ErrInvalidAccount)
}
