package player_created_subscriber_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/grant_sign_in_refill_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/subscribers/player_created_subscriber"
)

const (
	guest = "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"
	other = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
)

func setUp() (player_created_subscriber.Subscriber, *bonuses.FakeWallet) {
	rules := bonuses.TestRules()
	rules.SignInRefill = true
	wallet := bonuses.NewFakeWallet()
	useCase := grant_sign_in_refill_usecase.New(wallet, func() bonuses.Rules { return rules })
	return player_created_subscriber.New(useCase), wallet
}

func TestANewPlayerEarnsARefill(t *testing.T) {
	subscriber, wallet := setUp()

	require.NoError(t, subscriber.Handle(t.Context(), &playerv1.PlayerCreated{AccountId: guest}))

	assert.True(t, wallet.Kept(guest).Refill)
	assert.False(t, wallet.Kept(other).Refill)
}

func TestAnEventThatNamesNoAccountIsRefused(t *testing.T) {
	subscriber, wallet := setUp()

	err := subscriber.Handle(t.Context(), &playerv1.PlayerCreated{AccountId: "nope"})

	require.ErrorIs(t, err, bonuses.ErrInvalidAccount)
	assert.Empty(t, wallet.Payments())
}
