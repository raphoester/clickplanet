package player_forgotten_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/inmemory_contribution_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/standings/usecases/forget_account_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/subscribers/player_forgotten_subscriber"
)

func TestADeletedAccountLeavesTheStandings(t *testing.T) {
	store := inmemory_contribution_store.New()
	account, err := standings.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	require.NoError(t, store.RecordTake(t.Context(), "season-0", standings.Take{Account: account, Country: "fr"}))

	err = player_forgotten_subscriber.New(forget_account_usecase.New(store)).Handle(t.Context(),
		&playerv1.PlayerForgotten{AccountId: account.String()})

	require.NoError(t, err)
	assert.True(t, store.Tally("season-0", account).Empty())
}

func TestAnEventWithNoAccountIsRefused(t *testing.T) {
	err := player_forgotten_subscriber.New(forget_account_usecase.New(inmemory_contribution_store.New())).
		Handle(t.Context(), &playerv1.PlayerForgotten{})

	assert.ErrorIs(t, err, standings.ErrInvalidAccount)
}
