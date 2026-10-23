package grant_charges_usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/grant_charges_usecase"
)

const account = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

func storage() *bonuses.MemoryStore {
	return bonuses.NewMemoryStore()
}

func grant(store *bonuses.MemoryStore) *grant_charges_usecase.UseCase {
	return grant_charges_usecase.New(bonuses.NewCharges(store,
		bonuses.ChargesConfig{SpreadClicks: 8, Enclosures: 3, EnclosureMaxTiles: 25, Shields: 12}))
}

func TestEveryKindCanBeGranted(t *testing.T) {
	charges := storage()

	out, err := grant(charges).Execute(t.Context(), grant_charges_usecase.In{
		Account: account,
		Grant:   bonuses.Held{Refill: true, Bomb: true, Enclosures: 2, SpreadClicks: 5, Shields: 7},
	})
	require.NoError(t, err)

	assert.Equal(t, bonuses.Holder(account), out.Holder)
	assert.True(t, out.Before.Empty())
	assert.Equal(t, bonuses.Held{Refill: true, Bomb: true, Enclosures: 2, SpreadClicks: 5, Shields: 7}, out.After)
	assert.Equal(t, out.After, charges.Kept(account))
	for _, kind := range bonuses.Kinds {
		assert.Positive(t, out.After.Count(kind), kind)
	}
}

func TestAGrantAddsToWhatIsHeld(t *testing.T) {
	charges := storage()
	charges.Keep(account, bonuses.Held{Enclosures: 1})

	out, err := grant(charges).Execute(t.Context(), grant_charges_usecase.In{
		Account: account,
		Grant:   bonuses.Held{Enclosures: 1},
	})
	require.NoError(t, err)

	assert.Equal(t, bonuses.Held{Enclosures: 1}, out.Before)
	assert.Equal(t, bonuses.Held{Enclosures: 2}, out.After)
}

func TestAGrantStopsAtWhatFits(t *testing.T) {
	out, err := grant(storage()).Execute(t.Context(), grant_charges_usecase.In{
		Account: account,
		Grant:   bonuses.Held{Enclosures: 10, SpreadClicks: 100},
	})
	require.NoError(t, err)

	assert.Equal(t, bonuses.Held{Enclosures: 3, SpreadClicks: 8}, out.After)
}

func TestTheAccountIsHeldAsTheClickTokenNamesIt(t *testing.T) {
	charges := storage()

	_, err := grant(charges).Execute(t.Context(), grant_charges_usecase.In{
		Account: "0B7E5B6C-8F3A-4D2E-9C1A-2F6D8E4B7A10",
		Grant:   bonuses.Held{Bomb: true},
	})
	require.NoError(t, err)

	assert.Equal(t, bonuses.Held{Bomb: true}, charges.Kept(account))
}

func TestARefusedGrantChangesNothing(t *testing.T) {
	for _, tc := range []struct {
		in   grant_charges_usecase.In
		want error
	}{
		{grant_charges_usecase.In{Account: "a-guest", Grant: bonuses.Held{Bomb: true}}, bonuses.ErrInvalidHolder},
		{grant_charges_usecase.In{Account: account}, bonuses.ErrNothingToGrant},
		{grant_charges_usecase.In{Account: account, Grant: bonuses.Held{Bomb: true, SpreadClicks: -1}}, bonuses.ErrNegativeGrant},
	} {
		charges := storage()

		_, err := grant(charges).Execute(t.Context(), tc.in)

		require.ErrorIs(t, err, tc.want)
		assert.True(t, charges.Kept(account).Empty())
	}
}
