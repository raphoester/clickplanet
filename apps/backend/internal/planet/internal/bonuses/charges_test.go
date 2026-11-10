package bonuses_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

func TestABombIsSpentOnceAndOnlyByItsHolder(t *testing.T) {
	store := bonuses.NewMemoryStore()
	charges := bonuses.NewCharges(store, bonuses.TestRules)
	_, err := charges.Grant(t.Context(), "alice", bonuses.KindBomb, 1)
	require.NoError(t, err)

	require.ErrorIs(t, charges.Spend(t.Context(), "bob", bonuses.Held.AfterBomb), bonuses.ErrNotHeld)
	require.NoError(t, charges.Spend(t.Context(), "alice", bonuses.Held.AfterBomb))
	require.ErrorIs(t, charges.Spend(t.Context(), "alice", bonuses.Held.AfterBomb), bonuses.ErrNotHeld)
	assert.Equal(t, bonuses.Held{}, store.Kept("alice"))
}

func TestACallerWithNoAccountHoldsNothing(t *testing.T) {
	charges := bonuses.NewCharges(bonuses.NewMemoryStore(), bonuses.TestRules)

	gain, err := charges.Grant(t.Context(), bonuses.NoHolder, bonuses.KindBomb, 1)
	require.NoError(t, err)
	assert.Equal(t, bonuses.Gain{}, gain)

	held, err := charges.Held(t.Context(), bonuses.NoHolder)
	require.NoError(t, err)
	assert.Equal(t, bonuses.Held{}, held)
	require.ErrorIs(t, charges.Spend(t.Context(), bonuses.NoHolder, bonuses.Held.AfterBomb), bonuses.ErrNotHeld)
}

func TestAGrantAnswersTheHandItChangedAndWhatItAdded(t *testing.T) {
	store := bonuses.NewMemoryStore()
	store.Keep("alice", bonuses.Held{Bomb: true, SpreadClicks: 7})
	charges := bonuses.NewCharges(store, bonuses.TestRules)

	gain, err := charges.Grant(t.Context(), "alice", bonuses.KindSpreadClicks, 4)
	require.NoError(t, err)

	assert.Equal(t, bonuses.Gain{
		Before: bonuses.Held{Bomb: true, SpreadClicks: 7},
		After:  bonuses.Held{Bomb: true, SpreadClicks: 8},
	}, gain)
	assert.Equal(t, 1, gain.Count(bonuses.KindSpreadClicks), "the pool holds 8")
	assert.Equal(t, store.Kept("alice"), gain.After)
}

func TestAGrantOfWhatIsHeldAlreadyAddsNothing(t *testing.T) {
	store := bonuses.NewMemoryStore()
	store.Keep("alice", bonuses.Held{Bomb: true})
	charges := bonuses.NewCharges(store, bonuses.TestRules)

	gain, err := charges.Grant(t.Context(), "alice", bonuses.KindBomb, 1)
	require.NoError(t, err)

	assert.Zero(t, gain.Count(bonuses.KindBomb))
}

func TestAFailedGrantAnswersNoHand(t *testing.T) {
	store := bonuses.NewMemoryStore()
	store.FailWith(errors.New("postgres is down"))
	charges := bonuses.NewCharges(store, bonuses.TestRules)

	gain, err := charges.Grant(t.Context(), "alice", bonuses.KindBomb, 1)
	require.Error(t, err)
	assert.Equal(t, bonuses.Gain{}, gain)
}
