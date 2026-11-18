package player_forgotten_subscriber_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames/inmemory_past_name_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames/usecases/forget_past_names_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/subscribers/player_forgotten_subscriber"
)

func TestAForgottenAccountsPastNamesGo(t *testing.T) {
	store := inmemory_past_name_store.New()
	account := players.AccountID{15: 1}
	name, err := pastnames.Snapshot{Account: account, Name: "BraveFox42", LeftAt: time.Now()}.Restore()
	require.NoError(t, err)
	require.NoError(t, store.Record(t.Context(), name))

	err = player_forgotten_subscriber.New(forget_past_names_usecase.New(store)).
		Handle(t.Context(), &playerv1.PlayerForgotten{AccountId: account.String()})

	require.NoError(t, err)
	assert.Empty(t, store.Kept())
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := player_forgotten_subscriber.New(forget_past_names_usecase.New(inmemory_past_name_store.New())).
		Handle(t.Context(), &playerv1.PlayerForgotten{})

	assert.ErrorIs(t, err, players.ErrInvalidAccount)
}
