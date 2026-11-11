package grant_charges_usecase_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges/usecases/grant_charges_usecase"
)

const account = "0b7e5b6c-8f3a-4d2e-9c1a-2f6d8e4b7a10"

func storage() *charges.MemoryStore {
	return charges.NewMemoryStore()
}

type recordingStreams struct {
	told map[charges.Holder]charges.Held
}

func (s *recordingStreams) PublishGranted(holder charges.Holder, held charges.Held) {
	s.told[holder] = held
}

func grantTelling(store *charges.MemoryStore, streams *recordingStreams) *grant_charges_usecase.UseCase {
	rules := charges.TestRules()
	rules.Shield.Held = 12
	return grant_charges_usecase.New(charges.NewCharges(store, func() charges.Rules { return rules }), streams)
}

func grant(store *charges.MemoryStore) *grant_charges_usecase.UseCase {
	return grantTelling(store, &recordingStreams{told: map[charges.Holder]charges.Held{}})
}

func TestTheAccountsStreamsAreToldAllItHoldsAfterTheGrant(t *testing.T) {
	wallet := storage()
	wallet.Keep(account, charges.Held{Enclosures: 1})
	streams := &recordingStreams{told: map[charges.Holder]charges.Held{}}

	_, err := grantTelling(wallet, streams).Execute(t.Context(), grant_charges_usecase.In{
		Account: account,
		Grant:   charges.Held{Bomb: true},
	})
	require.NoError(t, err)

	assert.Equal(t, map[charges.Holder]charges.Held{account: {Bomb: true, Enclosures: 1}}, streams.told)
}

func TestEveryKindCanBeGranted(t *testing.T) {
	wallet := storage()

	out, err := grant(wallet).Execute(t.Context(), grant_charges_usecase.In{
		Account: account,
		Grant:   charges.Held{Refill: true, Bomb: true, Enclosures: 2, SpreadClicks: 5, Shields: 7},
	})
	require.NoError(t, err)

	assert.Equal(t, charges.Holder(account), out.Holder)
	assert.True(t, out.Before.Empty())
	assert.Equal(t, charges.Held{Refill: true, Bomb: true, Enclosures: 2, SpreadClicks: 5, Shields: 7}, out.After)
	assert.Equal(t, out.After, wallet.Kept(account))
	for _, kind := range charges.Kinds {
		assert.Positive(t, out.After.Count(kind), kind)
	}
}

func TestAGrantAddsToWhatIsHeld(t *testing.T) {
	wallet := storage()
	wallet.Keep(account, charges.Held{Enclosures: 1})

	out, err := grant(wallet).Execute(t.Context(), grant_charges_usecase.In{
		Account: account,
		Grant:   charges.Held{Enclosures: 1},
	})
	require.NoError(t, err)

	assert.Equal(t, charges.Held{Enclosures: 1}, out.Before)
	assert.Equal(t, charges.Held{Enclosures: 2}, out.After)
}

func TestAGrantStopsAtWhatFits(t *testing.T) {
	out, err := grant(storage()).Execute(t.Context(), grant_charges_usecase.In{
		Account: account,
		Grant:   charges.Held{Enclosures: 10, SpreadClicks: 100},
	})
	require.NoError(t, err)

	assert.Equal(t, charges.Held{Enclosures: 3, SpreadClicks: 8}, out.After)
}

func TestTheAccountIsHeldAsTheClickTokenNamesIt(t *testing.T) {
	wallet := storage()

	_, err := grant(wallet).Execute(t.Context(), grant_charges_usecase.In{
		Account: "0B7E5B6C-8F3A-4D2E-9C1A-2F6D8E4B7A10",
		Grant:   charges.Held{Bomb: true},
	})
	require.NoError(t, err)

	assert.Equal(t, charges.Held{Bomb: true}, wallet.Kept(account))
}

func TestARefusedGrantChangesNothing(t *testing.T) {
	for _, tc := range []struct {
		in   grant_charges_usecase.In
		want error
	}{
		{grant_charges_usecase.In{Account: "a-guest", Grant: charges.Held{Bomb: true}}, charges.ErrInvalidHolder},
		{grant_charges_usecase.In{Account: account}, charges.ErrNothingToGrant},
		{grant_charges_usecase.In{Account: account, Grant: charges.Held{Bomb: true, SpreadClicks: -1}}, charges.ErrNegativeGrant},
	} {
		wallet := storage()

		_, err := grant(wallet).Execute(t.Context(), tc.in)

		require.ErrorIs(t, err, tc.want)
		assert.True(t, wallet.Kept(account).Empty())
	}
}
