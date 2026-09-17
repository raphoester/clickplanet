package account_deleted_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/account_deleted_subscriber"
)

func TestADeletedAccountIsForgotten(t *testing.T) {
	store := inmemory_player_store.New()
	account, err := players.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	require.NoError(t, store.RecordTake(t.Context(), account, time.Now()))

	err = account_deleted_subscriber.New(forget_account_usecase.New(store)).Handle(t.Context(),
		&authv1.AccountDeleted{AccountId: account.String()})

	require.NoError(t, err)
	_, err = store.Stats(t.Context(), account)
	assert.ErrorIs(t, err, players.ErrNoStats)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	store := inmemory_player_store.New()

	err := account_deleted_subscriber.New(forget_account_usecase.New(store)).Handle(t.Context(), &authv1.AccountDeleted{})

	assert.ErrorIs(t, err, players.ErrInvalidAccount)
}
