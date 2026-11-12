package grant_sign_in_refill_usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/grant_sign_in_refill_usecase"
)

const (
	ada   = bonuses.Account("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	grace = bonuses.Account("7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f")
)

func setUp(signInRefill bool) (*grant_sign_in_refill_usecase.UseCase, *bonuses.FakeWallet) {
	rules := bonuses.TestRules()
	rules.SignInRefill = signInRefill
	wallet := bonuses.NewFakeWallet()
	return grant_sign_in_refill_usecase.New(wallet, func() bonuses.Rules { return rules }), wallet
}

func TestTheRefillIsPaidUnderTheAccountsOwnGrantSoTheWalletPaysItOnce(t *testing.T) {
	useCase, wallet := setUp(true)
	require.NoError(t, useCase.Execute(t.Context(), ada))
	wallet.Keep(ada, bonuses.Hand{})

	require.NoError(t, useCase.Execute(t.Context(), ada))

	assert.False(t, wallet.Kept(ada).Refill, "told again after it was used, it earns nothing")
	payments := wallet.Payments()
	require.Len(t, payments, 2)
	assert.Equal(t, bonuses.Payment{Grant: bonuses.SignInGrantOf(ada), Account: ada, Kind: bonuses.KindRefill, Amount: 1},
		payments[0])
	assert.Equal(t, payments[0], payments[1])
}

func TestEachAccountEarnsItsOwn(t *testing.T) {
	useCase, wallet := setUp(true)

	require.NoError(t, useCase.Execute(t.Context(), ada))
	require.NoError(t, useCase.Execute(t.Context(), grace))

	assert.True(t, wallet.Kept(ada).Refill)
	assert.True(t, wallet.Kept(grace).Refill)
}

func TestNoRefillWhileTheRuleIsOff(t *testing.T) {
	useCase, wallet := setUp(false)

	require.NoError(t, useCase.Execute(t.Context(), ada))

	assert.Empty(t, wallet.Payments())
}

func TestTheGrantIDIsTheOnePlanetKept(t *testing.T) {
	// charge_grants holds the refills paid before the cut: the id of an account may not change.
	assert.Equal(t, "5b2c9d77-1415-5970-bcb5-9e732480be86", bonuses.SignInGrantOf(ada).String())
}
