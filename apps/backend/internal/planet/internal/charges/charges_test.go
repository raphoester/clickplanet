package charges_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
)

func TestABombIsSpentOnceAndOnlyByItsHolder(t *testing.T) {
	store := charges.NewMemoryStore()
	wallet := charges.NewCharges(store, charges.TestRules)
	_, err := wallet.Grant(t.Context(), "alice", charges.KindBomb, 1)
	require.NoError(t, err)

	require.ErrorIs(t, wallet.Spend(t.Context(), "bob", charges.Held.AfterBomb), charges.ErrNotHeld)
	require.NoError(t, wallet.Spend(t.Context(), "alice", charges.Held.AfterBomb))
	require.ErrorIs(t, wallet.Spend(t.Context(), "alice", charges.Held.AfterBomb), charges.ErrNotHeld)
	assert.Equal(t, charges.Held{}, store.Kept("alice"))
}

func TestAChargeReturnedComesBackAboveTheStackSize(t *testing.T) {
	store := charges.NewMemoryStore()
	store.Keep("alice", charges.Held{Shields: 31})
	wallet := charges.NewCharges(store, charges.TestRules)

	require.NoError(t, wallet.Spend(t.Context(), "alice", charges.Held.AfterShield))
	require.NoError(t, wallet.Return(t.Context(), "alice", charges.Held.WithShieldBack))

	assert.Equal(t, charges.Held{Shields: 31}, store.Kept("alice"))
}

func TestACallerWithNoAccountHoldsNothing(t *testing.T) {
	wallet := charges.NewCharges(charges.NewMemoryStore(), charges.TestRules)

	gain, err := wallet.Grant(t.Context(), charges.NoHolder, charges.KindBomb, 1)
	require.NoError(t, err)
	assert.Equal(t, charges.Gain{}, gain)

	held, err := wallet.Held(t.Context(), charges.NoHolder)
	require.NoError(t, err)
	assert.Equal(t, charges.Held{}, held)
	require.ErrorIs(t, wallet.Spend(t.Context(), charges.NoHolder, charges.Held.AfterBomb), charges.ErrNotHeld)
}

func TestAGrantAnswersTheHandItChangedAndWhatItAdded(t *testing.T) {
	store := charges.NewMemoryStore()
	store.Keep("alice", charges.Held{Bomb: true, SpreadClicks: 7})
	wallet := charges.NewCharges(store, charges.TestRules)

	gain, err := wallet.Grant(t.Context(), "alice", charges.KindSpreadClicks, 4)
	require.NoError(t, err)

	assert.Equal(t, charges.Gain{
		Before: charges.Held{Bomb: true, SpreadClicks: 7},
		After:  charges.Held{Bomb: true, SpreadClicks: 8},
	}, gain)
	assert.Equal(t, 1, gain.Count(charges.KindSpreadClicks), "the pool holds 8")
	assert.Equal(t, store.Kept("alice"), gain.After)
}

func TestAGrantOfWhatIsHeldAlreadyAddsNothing(t *testing.T) {
	store := charges.NewMemoryStore()
	store.Keep("alice", charges.Held{Bomb: true})
	wallet := charges.NewCharges(store, charges.TestRules)

	gain, err := wallet.Grant(t.Context(), "alice", charges.KindBomb, 1)
	require.NoError(t, err)

	assert.Zero(t, gain.Count(charges.KindBomb))
}

func TestAFailedGrantAnswersNoHand(t *testing.T) {
	store := charges.NewMemoryStore()
	store.FailWith(errors.New("postgres is down"))
	wallet := charges.NewCharges(store, charges.TestRules)

	gain, err := wallet.Grant(t.Context(), "alice", charges.KindBomb, 1)
	require.Error(t, err)
	assert.Equal(t, charges.Gain{}, gain)
}
