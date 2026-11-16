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
	useCase, wallet, _ := setUpOn(bonuses.TestRules(), kinds)
	return useCase, wallet
}

// setUpOn plays rules with these weights; nobody has a kind unlocked until the test unlocks one.
func setUpOn(
	rules bonuses.Rules, kinds map[bonuses.Kind]float64,
) (*pay_right_answer_usecase.UseCase, *bonuses.FakeWallet, *bonuses.FakeUnlocks) {
	rules.Kinds = kinds
	wallet, unlocks := bonuses.NewFakeWallet(), bonuses.NewFakeUnlocks()
	return pay_right_answer_usecase.New(wallet, unlocks, func() bonuses.Rules { return rules }), wallet, unlocks
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

	assert.Zero(t, wallet.Kept(ada).Refills, "the wallet pays each grant once")
}

func TestAKindDrawnIntoAFullStackIsKeptNoneOf(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindRefill: 1, bonuses.KindShields: 0})
	wallet.Size(bonuses.KindRefill, 1)
	wallet.Keep(ada, bonuses.Hand{Refills: 1})

	require.NoError(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token"}))

	require.Len(t, wallet.Payments(), 1, "the draw ignores the hand")
	assert.Equal(t, bonuses.KindRefill, wallet.Payments()[0].Kind)
	assert.Equal(t, bonuses.Hand{Refills: 1}, wallet.Kept(ada))
}

func TestANewcomersFirstQuizPaysShields(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindRefill: 100, bonuses.KindShields: 1})

	require.NoError(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token", Welcome: true}))

	require.Len(t, wallet.Payments(), 1)
	assert.Equal(t, bonuses.KindShields, wallet.Payments()[0].Kind)
}

func TestAGuestsFirstQuizPaysShieldsThoughTheGatesCloseThem(t *testing.T) {
	rules := bonuses.TestGated()
	rules.Gates.Open = map[bonuses.Kind]bool{bonuses.KindRefill: true}
	useCase, wallet, _ := setUpOn(rules, map[bonuses.Kind]float64{bonuses.KindRefill: 100, bonuses.KindShields: 1})

	require.NoError(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token", Welcome: true}))

	require.Len(t, wallet.Payments(), 1)
	assert.Equal(t, bonuses.KindShields, wallet.Payments()[0].Kind)
}

func TestANewcomerWithAFullStackOfShieldsIsPaidNothing(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindRefill: 1, bonuses.KindShields: 1})
	wallet.Size(bonuses.KindShields, 10)
	wallet.Keep(ada, bonuses.Hand{Shields: 10})

	require.NoError(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token", Welcome: true}))

	require.Len(t, wallet.Payments(), 1)
	assert.Equal(t, bonuses.KindShields, wallet.Payments()[0].Kind, "never a refill or a bomb for a newcomer")
	assert.Equal(t, bonuses.Hand{Shields: 10}, wallet.Kept(ada))
}

func TestAPaymentTheWalletCannotMakeIsTriedAgain(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindRefill: 1})
	wallet.FailWith(errors.New("planet is gone"))

	require.Error(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token"}))
}

func TestWithTheGatesOnARightAnswerIsDrawnAmongTheOpenAndUnlockedKinds(t *testing.T) {
	useCase, wallet, unlocks := setUpOn(bonuses.TestGated(), map[bonuses.Kind]float64{
		bonuses.KindBomb: 100, bonuses.KindEncloseClicks: 1,
	})
	unlocks.Unlock(ada, bonuses.KindEncloseClicks)

	for range 20 {
		require.NoError(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token"}))
	}

	for _, payment := range wallet.Payments() {
		require.Equal(t, bonuses.KindEncloseClicks, payment.Kind, "progression has not unlocked the bomb")
	}
}

func TestWithTheGatesOnAGuestsRightAnswerIsDrawnOnlyTheOpenKinds(t *testing.T) {
	useCase, wallet, _ := setUpOn(bonuses.TestGated(), map[bonuses.Kind]float64{bonuses.KindBomb: 1})

	require.NoError(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token"}))

	assert.Empty(t, wallet.Payments(), "the open kinds weigh 0 here, and a guest has the bomb unlocked nowhere")
}

func TestARightAnswerWhoseUnlocksCannotBeReadIsTriedAgain(t *testing.T) {
	useCase, wallet, unlocks := setUpOn(bonuses.TestRules(), map[bonuses.Kind]float64{bonuses.KindRefill: 1})
	unlocks.FailWith(errors.New("postgres is gone"))

	require.Error(t, useCase.Execute(t.Context(), bonuses.Answer{Account: ada, Token: "a-token"}))
	assert.Empty(t, wallet.Payments())
}
