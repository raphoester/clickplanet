package pay_level_up_usecase_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/pay_level_up_usecase"
)

const ada = bonuses.Account("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

var grant = bonuses.GrantID(uuid.MustParse("6a1f3c2e-8d4b-4f7a-9e21-5b3c7d9e0f11"))

func setUp(rules bonuses.Rules, kinds map[bonuses.Kind]float64) (*pay_level_up_usecase.UseCase, *bonuses.FakeWallet) {
	rules.Kinds = kinds
	wallet := bonuses.NewFakeWallet()
	return pay_level_up_usecase.New(wallet, func() bonuses.Rules { return rules }), wallet
}

func TestALevelUpIsPaidOneBonusUnderProgressionsGrant(t *testing.T) {
	useCase, wallet := setUp(bonuses.TestRules(), map[bonuses.Kind]float64{bonuses.KindRefill: 1})

	require.NoError(t, useCase.Execute(t.Context(), bonuses.LevelUp{Account: ada, Grant: grant}))

	payments := wallet.Payments()
	require.Len(t, payments, 1)
	assert.Equal(t, grant, payments[0].Grant)
	assert.Equal(t, 1, wallet.Kept(ada).Refills)
}

func TestALevelUpToldAgainIsPaidOnce(t *testing.T) {
	useCase, wallet := setUp(bonuses.TestRules(), map[bonuses.Kind]float64{bonuses.KindRefill: 1})
	levelUp := bonuses.LevelUp{Account: ada, Grant: grant}

	require.NoError(t, useCase.Execute(t.Context(), levelUp))
	wallet.Keep(ada, bonuses.Hand{})
	require.NoError(t, useCase.Execute(t.Context(), levelUp))

	assert.Zero(t, wallet.Kept(ada).Refills, "the wallet pays each grant once")
}

func TestWithTheGatesOnALevelUpIsDrawnAmongTheOpenAndUnlockedKinds(t *testing.T) {
	useCase, wallet := setUp(bonuses.TestGated(), map[bonuses.Kind]float64{bonuses.KindBomb: 100, bonuses.KindEncloseClicks: 1})

	for range 20 {
		require.NoError(t, useCase.Execute(t.Context(), bonuses.LevelUp{
			Account: ada, Grant: grant, Unlocked: bonuses.Unlocked{bonuses.KindEncloseClicks},
		}))
		wallet.Keep(ada, bonuses.Hand{})
	}

	for _, payment := range wallet.Payments() {
		require.Equal(t, bonuses.KindEncloseClicks, payment.Kind, "the level unlocks no bomb")
	}
}

func TestALevelUpWithNothingToDrawIsPaidNothing(t *testing.T) {
	useCase, wallet := setUp(bonuses.TestGated(), map[bonuses.Kind]float64{bonuses.KindBomb: 1})

	require.NoError(t, useCase.Execute(t.Context(), bonuses.LevelUp{Account: ada, Grant: grant}))

	assert.Empty(t, wallet.Payments(), "the bomb is not unlocked, and nothing else weighs")
}

func TestAPaymentTheWalletCannotMakeIsTriedAgain(t *testing.T) {
	useCase, wallet := setUp(bonuses.TestRules(), map[bonuses.Kind]float64{bonuses.KindRefill: 1})
	wallet.FailWith(errors.New("planet is gone"))

	require.Error(t, useCase.Execute(t.Context(), bonuses.LevelUp{Account: ada, Grant: grant}))
}
