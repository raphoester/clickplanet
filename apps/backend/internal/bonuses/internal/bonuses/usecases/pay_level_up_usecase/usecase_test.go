package pay_level_up_usecase_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses/usecases/pay_level_up_usecase"
)

const ada = bonuses.Account("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

var grant = bonuses.GrantID(uuid.MustParse("6a1f3c2e-8d4b-4f7a-9e21-5b3c7d9e0f11"))

func setUp(rules bonuses.Rules) (*pay_level_up_usecase.UseCase, *bonuses.FakeWallet) {
	wallet := bonuses.NewFakeWallet()
	wallet.Size(bonuses.KindRefill, 1)
	wallet.Size(bonuses.KindSpreadClicks, 8)
	wallet.Size(bonuses.KindBomb, 1)
	wallet.Size(bonuses.KindEncloseClicks, 3)
	wallet.Size(bonuses.KindShields, 30)
	return pay_level_up_usecase.New(wallet, func() bonuses.Rules { return rules }), wallet
}

func TestALevelUpFillsEveryStackUnderProgressionsGrant(t *testing.T) {
	useCase, wallet := setUp(bonuses.TestRules())
	wallet.Keep(ada, bonuses.Hand{Shields: 12, SpreadClicks: 8})

	require.NoError(t, useCase.Execute(t.Context(), bonuses.LevelUp{Account: ada, Grant: grant}))

	fills := wallet.Fills()
	require.Len(t, fills, 1, "one grant, so one notice on the player's stream")
	assert.Equal(t, grant, fills[0].Grant)
	assert.Empty(t, wallet.Payments(), "no bonus drawn")
	assert.Equal(t, bonuses.Hand{Refills: 1, SpreadClicks: 8, Bombs: 1, Enclosures: 3, Shields: 30}, wallet.Kept(ada))
}

func TestALevelUpToldAgainIsFilledOnce(t *testing.T) {
	useCase, wallet := setUp(bonuses.TestRules())
	levelUp := bonuses.LevelUp{Account: ada, Grant: grant}

	require.NoError(t, useCase.Execute(t.Context(), levelUp))
	wallet.Keep(ada, bonuses.Hand{})
	require.NoError(t, useCase.Execute(t.Context(), levelUp))

	assert.Equal(t, bonuses.Hand{}, wallet.Kept(ada), "the wallet fills each grant once")
}

func TestWithTheGatesOnALevelUpFillsOnlyTheOpenAndUnlockedKinds(t *testing.T) {
	useCase, wallet := setUp(bonuses.TestGated())

	require.NoError(t, useCase.Execute(t.Context(), bonuses.LevelUp{
		Account: ada, Grant: grant, Unlocked: bonuses.Unlocked{bonuses.KindEncloseClicks},
	}))

	assert.Zero(t, wallet.Kept(ada).Bombs, "the level unlocks no bomb")
	assert.Equal(t, 3, wallet.Kept(ada).Enclosures)
}

func TestALevelUpWithNothingToFillAsksTheWalletNothing(t *testing.T) {
	rules := bonuses.TestGated()
	rules.Kinds = map[bonuses.Kind]float64{bonuses.KindBomb: 1}
	useCase, wallet := setUp(rules)

	require.NoError(t, useCase.Execute(t.Context(), bonuses.LevelUp{Account: ada, Grant: grant}))

	assert.Empty(t, wallet.Fills(), "the bomb is not unlocked, and nothing else weighs")
}
