package planet

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/inmemory_grant_store"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/pay_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const ada = bonuses.Account("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")

func walletOver(hands *charges.MemoryStore) wallet {
	rules := charges.TestRules()
	rules.Enclose.Held, rules.Shield.Held = 3, 30
	inForce := func() charges.Rules { return rules }
	charged := charges.NewCharges(hands, inForce)

	return newWallet(charged, inForce, pay_usecase.New(cpatomic.FakeTransactioner{}, inmemory_grant_store.New(time.Minute),
		charged, cptime.NewFixedClock(time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC))))
}

// The kinds cross the seam by name: a kind one side names and the other does not would be paid as nothing.
func TestTheOffersAndTheWalletNameTheSameKinds(t *testing.T) {
	require.Len(t, charges.Kinds, len(bonuses.Kinds))
	for i, kind := range bonuses.Kinds {
		assert.Equal(t, string(kind), string(charges.Kinds[i]))
	}
}

func TestTheRoomIsTheKindsEachAccountHoldsAsManyOfAsTheWalletAllows(t *testing.T) {
	hands := charges.NewMemoryStore()
	hands.Keep(charges.Holder(ada), charges.Held{Bomb: true, Enclosures: 3, Shields: 29})

	full, err := walletOver(hands).Full(t.Context(), []bonuses.Account{ada, "another"})
	require.NoError(t, err)

	assert.ElementsMatch(t, []bonuses.Kind{bonuses.KindBomb, bonuses.KindEncloseClicks}, full[ada])
	assert.Empty(t, full["another"])
}

func TestAHandTheWalletCannotReadIsAnError(t *testing.T) {
	hands := charges.NewMemoryStore()
	hands.FailWith(errors.New("postgres is down"))

	_, err := walletOver(hands).Full(t.Context(), []bonuses.Account{ada})

	require.Error(t, err)
}

func TestAPaymentIsGrantedOnceAndAnswersWhatWasKept(t *testing.T) {
	hands := charges.NewMemoryStore()
	hands.Keep(charges.Holder(ada), charges.Held{Enclosures: 2})
	payment := bonuses.Payment{Grant: bonuses.OfferGrantOf("a-token"), Account: ada, Kind: bonuses.KindEncloseClicks, Amount: 3}
	w := walletOver(hands)

	paid, err := w.Pay(t.Context(), payment)
	require.NoError(t, err)
	again, err := w.Pay(t.Context(), payment)
	require.NoError(t, err)

	assert.Equal(t, bonuses.Paid{Kept: 1, Hand: bonuses.Hand{Enclosures: 3}}, paid, "the stack holds 3")
	assert.Equal(t, bonuses.Paid{Hand: bonuses.Hand{Enclosures: 3}}, again, "told again, it keeps nothing")
	hand, err := w.Hand(t.Context(), ada)
	require.NoError(t, err)
	assert.Equal(t, bonuses.Hand{Enclosures: 3}, hand)
}
