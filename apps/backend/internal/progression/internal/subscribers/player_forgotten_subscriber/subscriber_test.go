package player_forgotten_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/inmemory_stats_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/stats/usecases/forget_stats_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/player_forgotten_subscriber"
)

func TestAForgottenPlayerLosesItsStats(t *testing.T) {
	store := inmemory_stats_store.New()
	ada := accounts.AccountID{15: 1}
	require.NoError(t, store.Record(t.Context(), ada, stats.TakeAt(0, 0, time.Now())))

	require.NoError(t, player_forgotten_subscriber.New(forget_stats_usecase.New(store)).Handle(t.Context(),
		&playerv1.PlayerForgotten{AccountId: ada.String()}))

	_, err := store.Stats(t.Context(), ada)
	assert.ErrorIs(t, err, stats.ErrNoStats)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := player_forgotten_subscriber.New(forget_stats_usecase.New(inmemory_stats_store.New())).Handle(t.Context(),
		&playerv1.PlayerForgotten{})

	assert.ErrorIs(t, err, accounts.ErrInvalidAccount)
}
