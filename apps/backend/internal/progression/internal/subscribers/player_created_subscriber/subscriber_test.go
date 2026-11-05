package player_created_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/nameplates/inmemory_nameplate_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/nameplates/usecases/keep_nameplate_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/subscribers/player_created_subscriber"
)

func TestANewPlayerGetsANameplate(t *testing.T) {
	store := inmemory_nameplate_store.New()
	ada := accounts.AccountID{15: 1}

	require.NoError(t, player_created_subscriber.New(keep_nameplate_usecase.New(store)).Handle(t.Context(),
		&playerv1.PlayerCreated{AccountId: ada.String()}))

	named, err := store.Named(t.Context(), ada)
	require.NoError(t, err)
	assert.True(t, named)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := player_created_subscriber.New(keep_nameplate_usecase.New(inmemory_nameplate_store.New())).Handle(t.Context(),
		&playerv1.PlayerCreated{})

	assert.ErrorIs(t, err, accounts.ErrInvalidAccount)
}
