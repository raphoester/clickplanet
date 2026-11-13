package pay_right_answer_usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/pay_right_answer_usecase"
)

const ada = bonuses.Account("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

func setUp(kinds map[bonuses.Kind]float64) (*pay_right_answer_usecase.UseCase, *bonuses.FakeWallet) {
	rules := bonuses.TestRules()
	rules.Kinds = kinds
	wallet := bonuses.NewFakeWallet()
	return pay_right_answer_usecase.New(wallet, wallet, func() bonuses.Rules { return rules }), wallet
}

func TestARightAnswerIsPaidAKindDrawnAsABoxsUnderItsQuizsToken(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindSpreadClicks: 1})

	require.NoError(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token"}))

	payments := wallet.Payments()
	require.Len(t, payments, 1)
	assert.Equal(t, bonuses.OfferGrantOf("a-token"), payments[0].Grant, "the grant the offers named it by before the cut")
	assert.Equal(t, bonuses.KindSpreadClicks, payments[0].Kind)
	assert.Positive(t, wallet.Kept(ada).SpreadClicks)
}

func TestAnAnswerToldAgainIsPaidOnce(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindRefill: 1})
	answer := bonuses.Answer{Account: ada, Token: "a-token"}

	require.NoError(t, useCase.Execute(t.Context(), answer))
	wallet.Keep(ada, bonuses.Hand{})
	require.NoError(t, useCase.Execute(t.Context(), answer))

	assert.False(t, wallet.Kept(ada).Refill, "the wallet pays each grant once")
}

func TestAKindTheHandHasNoRoomForIsNotDrawn(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindRefill: 100, bonuses.KindShields: 1})
	wallet.Keep(ada, bonuses.Hand{Refill: true})

	require.NoError(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token"}))

	require.Len(t, wallet.Payments(), 1)
	assert.Equal(t, bonuses.KindShields, wallet.Payments()[0].Kind)
}

func TestAHandWithRoomForNothingIsPaidNothing(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindRefill: 1})
	wallet.Keep(ada, bonuses.Hand{Refill: true})

	require.NoError(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token"}))

	assert.Empty(t, wallet.Payments())
}

func TestANewcomersFirstQuizPaysShields(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindRefill: 100, bonuses.KindShields: 1})

	require.NoError(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token", Welcome: true}))

	require.Len(t, wallet.Payments(), 1)
	assert.Equal(t, bonuses.KindShields, wallet.Payments()[0].Kind)
}

func TestANewcomerWithAFullStackOfShieldsIsPaidNothing(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindRefill: 1, bonuses.KindShields: 1})
	wallet.Size(bonuses.KindShields, 10)
	wallet.Keep(ada, bonuses.Hand{Shields: 10})

	require.NoError(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token", Welcome: true}))

	assert.Empty(t, wallet.Payments(), "never a refill or a bomb for a newcomer")
}

func TestARoomTheWalletCannotTellIsTriedAgain(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindRefill: 1})
	wallet.FailWith(errors.New("planet is gone"))

	require.Error(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token"}))
}
