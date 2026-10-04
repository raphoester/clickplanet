package account_deleted_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/messages"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/inmemory_seen_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/seen/usecases/forget_seen_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/subscribers/account_deleted_subscriber"
)

var at = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

func TestADeletedAccountsSeenMarkIsForgotten(t *testing.T) {
	store := inmemory_seen_storage.New()
	ada, bob := messages.AccountID{15: 1}, messages.AccountID{15: 2}
	require.NoError(t, store.SaveSeen(t.Context(), ada, at))
	require.NoError(t, store.SaveSeen(t.Context(), bob, at))

	err := account_deleted_subscriber.New(forget_seen_usecase.New(store)).Handle(t.Context(),
		&authv1.AccountDeleted{AccountId: ada.String()})

	require.NoError(t, err)
	assert.True(t, store.Kept(ada).IsZero())
	assert.Equal(t, at, store.Kept(bob))
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := account_deleted_subscriber.New(forget_seen_usecase.New(inmemory_seen_storage.New())).Handle(t.Context(),
		&authv1.AccountDeleted{})

	assert.Error(t, err)
}
