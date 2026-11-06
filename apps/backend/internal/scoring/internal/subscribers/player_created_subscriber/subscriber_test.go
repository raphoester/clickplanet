package player_created_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/players/inmemory_player_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/players/usecases/keep_player_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/player_created_subscriber"
)

func TestANewPlayerIsKeptAsOne(t *testing.T) {
	store := inmemory_player_store.New()
	ada := standings.AccountID{15: 1}

	require.NoError(t, player_created_subscriber.New(keep_player_usecase.New(store)).Handle(t.Context(),
		&playerv1.PlayerCreated{AccountId: ada.String()}))

	is, err := store.IsPlayer(t.Context(), ada)
	require.NoError(t, err)
	assert.True(t, is)
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := player_created_subscriber.New(keep_player_usecase.New(inmemory_player_store.New())).Handle(t.Context(),
		&playerv1.PlayerCreated{})

	assert.ErrorIs(t, err, standings.ErrInvalidAccount)
}
