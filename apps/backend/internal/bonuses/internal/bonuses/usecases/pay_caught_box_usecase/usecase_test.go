package pay_caught_box_usecase_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/pay_caught_box_usecase"
)

const ada = bonuses.Account("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

func setUp(kinds map[bonuses.Kind]float64) (*pay_caught_box_usecase.UseCase, *bonuses.FakeWallet) {
	useCase, wallet, _ := setUpOn(bonuses.TestRules(), kinds)
	return useCase, wallet
}

// setUpOn plays rules with these weights; nobody has a kind unlocked until the test unlocks one.
func setUpOn(
	rules bonuses.Rules, kinds map[bonuses.Kind]float64,
) (*pay_caught_box_usecase.UseCase, *bonuses.FakeWallet, *bonuses.FakeUnlocks) {
	rules.Kinds = kinds
	wallet, unlocks := bonuses.NewFakeWallet(), bonuses.NewFakeUnlocks()
	return pay_caught_box_usecase.New(wallet, unlocks, func() bonuses.Rules { return rules }), wallet, unlocks
}

func TestABoxCaughtIsPaidAKindDrawnFromTheWeightsUnderItsToken(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindEncloseClicks: 1})

	out, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token"})
	require.NoError(t, err)

	assert.True(t, out.Paid)
	payments := wallet.Payments()
	require.Len(t, payments, 1)
	assert.Equal(t, bonuses.OfferGrantOf("a-token"), payments[0].Grant, "the grant the box was paid under before the cut")
	assert.Equal(t, bonuses.KindEncloseClicks, payments[0].Kind)
	assert.Positive(t, wallet.Kept(ada).Enclosures)
}

func TestEveryKindWeighedIsPaidAsItself(t *testing.T) {
	for _, kind := range bonuses.Kinds {
		useCase, wallet := setUp(map[bonuses.Kind]float64{kind: 1})

		_, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token"})
		require.NoError(t, err)

		require.Len(t, wallet.Payments(), 1)
		assert.Equal(t, kind, wallet.Payments()[0].Kind)
		assert.GreaterOrEqual(t, wallet.Payments()[0].Amount, 1)
	}
}

func TestABoxToldAgainIsPaidOnce(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindRefill: 1})
	box := bonuses.CaughtBox{Account: ada, Token: "a-token"}

	_, err := useCase.Execute(t.Context(), box)
	require.NoError(t, err)
	wallet.Keep(ada, bonuses.Hand{})
	_, err = useCase.Execute(t.Context(), box)
	require.NoError(t, err)

	assert.Zero(t, wallet.Kept(ada).Refills, "the wallet pays each grant once")
}

func TestTheDrawIgnoresTheHand(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindBomb: 1, bonuses.KindRefill: 1})
	wallet.Size(bonuses.KindRefill, 1)
	wallet.Keep(ada, bonuses.Hand{Refills: 1})

	for i := range 60 {
		_, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: fmt.Sprintf("token-%d", i)})
		require.NoError(t, err)
	}

	drawn := map[bonuses.Kind]int{}
	for _, payment := range wallet.Payments() {
		drawn[payment.Kind]++
	}
	assert.Positive(t, drawn[bonuses.KindRefill], "a full stack never steers the draw to another kind")
	assert.Positive(t, drawn[bonuses.KindBomb])
}

func TestAKindDrawnIntoAFullStackIsPaidNothing(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindBomb: 1, bonuses.KindShields: 0})
	wallet.Size(bonuses.KindBomb, 1)
	wallet.Keep(ada, bonuses.Hand{Bombs: 1})

	out, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token"})
	require.NoError(t, err)

	assert.False(t, out.Paid, "the wallet kept none of it")
	require.Len(t, wallet.Payments(), 1)
	assert.Equal(t, bonuses.KindBomb, wallet.Payments()[0].Kind)
	assert.Equal(t, bonuses.Hand{Bombs: 1}, wallet.Kept(ada))
}

func TestANewcomersFirstBoxPaysASpread(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindRefill: 100, bonuses.KindSpreadClicks: 1})

	_, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token", Welcome: true})
	require.NoError(t, err)

	require.Len(t, wallet.Payments(), 1)
	assert.Equal(t, bonuses.KindSpreadClicks, wallet.Payments()[0].Kind)
}

func TestAGuestsFirstBoxPaysASpreadThoughTheGatesCloseIt(t *testing.T) {
	rules := bonuses.TestGated()
	rules.Gates.Open = map[bonuses.Kind]bool{bonuses.KindRefill: true}
	useCase, wallet, _ := setUpOn(rules, map[bonuses.Kind]float64{bonuses.KindRefill: 100, bonuses.KindSpreadClicks: 1})

	out, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token", Welcome: true})
	require.NoError(t, err)

	assert.True(t, out.Paid)
	require.Len(t, wallet.Payments(), 1)
	assert.Equal(t, bonuses.KindSpreadClicks, wallet.Payments()[0].Kind)
}

func TestANewcomersFirstBoxWithNoSpreadToPayIsPaidNothing(t *testing.T) {
	for _, kinds := range []map[bonuses.Kind]float64{
		{bonuses.KindRefill: 1, bonuses.KindBomb: 1},
		{bonuses.KindRefill: 1, bonuses.KindSpreadClicks: 1},
	} {
		useCase, wallet := setUp(kinds)
		wallet.Size(bonuses.KindSpreadClicks, 8)
		if kinds[bonuses.KindSpreadClicks] > 0 {
			wallet.Keep(ada, bonuses.Hand{SpreadClicks: 8})
		}

		out, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token", Welcome: true})
		require.NoError(t, err)

		assert.False(t, out.Paid)
		for _, payment := range wallet.Payments() {
			assert.Equal(t, bonuses.KindSpreadClicks, payment.Kind, "never a refill or a bomb for a newcomer")
		}
	}
}

func TestAPaymentTheWalletCannotMakeIsTriedAgain(t *testing.T) {
	useCase, wallet := setUp(map[bonuses.Kind]float64{bonuses.KindRefill: 1})
	wallet.FailWith(errors.New("planet is gone"))

	_, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token"})

	require.Error(t, err)
}

func TestWithTheGatesOnAKindNotUnlockedIsNeverDrawn(t *testing.T) {
	useCase, wallet, unlocks := setUpOn(bonuses.TestGated(), map[bonuses.Kind]float64{
		bonuses.KindBomb: 100, bonuses.KindEncloseClicks: 100, bonuses.KindRefill: 1,
	})
	unlocks.Unlock(ada, bonuses.KindEncloseClicks)

	for range 30 {
		_, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token"})
		require.NoError(t, err)
	}

	for _, payment := range wallet.Payments() {
		require.NotEqual(t, bonuses.KindBomb, payment.Kind, "progression has not unlocked the bomb")
	}
}

func TestWithTheGatesOnAKindUnlockedIsDrawn(t *testing.T) {
	useCase, wallet, unlocks := setUpOn(bonuses.TestGated(), map[bonuses.Kind]float64{bonuses.KindBomb: 1})
	unlocks.Unlock(ada, bonuses.KindEncloseClicks, bonuses.KindBomb)

	out, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token"})
	require.NoError(t, err)

	assert.True(t, out.Paid)
	assert.Equal(t, 1, wallet.Kept(ada).Bombs)
}

func TestWithTheGatesOnAGuestIsDrawnOnlyTheOpenKinds(t *testing.T) {
	useCase, wallet, _ := setUpOn(bonuses.TestGated(), map[bonuses.Kind]float64{
		bonuses.KindBomb: 100, bonuses.KindEncloseClicks: 100, bonuses.KindShields: 1,
	})

	for range 20 {
		_, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token"})
		require.NoError(t, err)
	}

	for _, payment := range wallet.Payments() {
		require.Equal(t, bonuses.KindShields, payment.Kind, "a guest has nothing unlocked")
	}
}

func TestWithTheGatesOffAGuestGetsABomb(t *testing.T) {
	useCase, wallet, _ := setUpOn(bonuses.TestRules(), map[bonuses.Kind]float64{bonuses.KindBomb: 1})

	_, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token"})
	require.NoError(t, err)

	assert.Equal(t, 1, wallet.Kept(ada).Bombs, "a battle's set keeps the gates off")
}

func TestABoxWhoseKindsItMayBeAllWeighZeroIsPaidNothing(t *testing.T) {
	useCase, wallet, unlocks := setUpOn(bonuses.TestGated(), map[bonuses.Kind]float64{bonuses.KindBomb: 1})
	unlocks.Unlock(ada, bonuses.KindEncloseClicks)

	out, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token"})
	require.NoError(t, err)

	assert.False(t, out.Paid)
	assert.Empty(t, wallet.Payments())
}

func TestTheChargesHeldStayWhateverTheGates(t *testing.T) {
	useCase, wallet, _ := setUpOn(bonuses.TestGated(), map[bonuses.Kind]float64{bonuses.KindRefill: 1})
	wallet.Keep(ada, bonuses.Hand{Bombs: 1, Enclosures: 3})

	_, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token"})
	require.NoError(t, err)

	assert.Equal(t, bonuses.Hand{Refills: 1, Bombs: 1, Enclosures: 3}, wallet.Kept(ada),
		"a gate decides what is drawn, never what is held")
}

func TestUnlocksThatCannotBeReadAreTriedAgain(t *testing.T) {
	useCase, wallet, unlocks := setUpOn(bonuses.TestRules(), map[bonuses.Kind]float64{bonuses.KindRefill: 1})
	unlocks.FailWith(errors.New("postgres is gone"))

	_, err := useCase.Execute(t.Context(), bonuses.CaughtBox{Account: ada, Token: "a-token"})

	require.Error(t, err)
	assert.Empty(t, wallet.Payments())
}
