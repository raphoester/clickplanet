package player_forgotten_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/inmemory_seen_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/usecases/forget_seen_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/player_forgotten_subscriber"
)

var at = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

func TestADeletedAccountsSeenMarkIsForgotten(t *testing.T) {
	store := inmemory_seen_storage.New()
	ada, bob := messages.AccountID{15: 1}, messages.AccountID{15: 2}
	require.NoError(t, store.SaveSeen(t.Context(), ada, messages.World, at))
	require.NoError(t, store.SaveSeen(t.Context(), ada, messages.FlagRoom("fr"), at))
	require.NoError(t, store.SaveSeen(t.Context(), bob, messages.World, at))

	err := player_forgotten_subscriber.New(forget_seen_usecase.New(store)).Handle(t.Context(),
		&playerv1.PlayerForgotten{AccountId: ada.String()})

	require.NoError(t, err)
	assert.True(t, store.Kept(ada, messages.World).IsZero())
	assert.True(t, store.Kept(ada, messages.FlagRoom("fr")).IsZero())
	assert.Equal(t, at, store.Kept(bob, messages.World))
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := player_forgotten_subscriber.New(forget_seen_usecase.New(inmemory_seen_storage.New())).Handle(t.Context(),
		&playerv1.PlayerForgotten{})

	assert.Error(t, err)
}
