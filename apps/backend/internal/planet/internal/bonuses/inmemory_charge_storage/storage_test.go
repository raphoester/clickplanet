package inmemory_charge_storage_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/inmemory_charge_storage"
)

const (
	alice bonuses.Holder = "alice"
	bob   bonuses.Holder = "bob"
)

var rules = bonuses.ChargesConfig{SpreadClicks: 8, Enclosures: 3, EnclosureMaxTiles: 25}

func loaded(t *testing.T, persistence inmemory_charge_storage.Persistence) *inmemory_charge_storage.Storage {
	t.Helper()

	storage := inmemory_charge_storage.New(inmemory_charge_storage.Config{}, rules, persistence, slog.New(slog.DiscardHandler))
	require.NoError(t, storage.Load(t.Context()))

	return storage
}

func fresh(t *testing.T) (*inmemory_charge_storage.Storage, *inmemory_charge_storage.MemoryPersistence) {
	t.Helper()

	persistence := inmemory_charge_storage.NewMemoryPersistence()

	return loaded(t, persistence), persistence
}

func TestAChargeBelongsToItsHolder(t *testing.T) {
	storage, _ := fresh(t)
	storage.Grant(alice, bonuses.KindBomb, 1)

	assert.False(t, storage.SpendBomb(bob))
	assert.Equal(t, bonuses.Held{}, storage.Held(bob))
	assert.True(t, storage.SpendBomb(alice))
	assert.False(t, storage.SpendBomb(alice), "a bomb that went off is gone")
}

func TestACallerWithNoAccountHoldsNothing(t *testing.T) {
	storage, persistence := fresh(t)

	storage.Grant(bonuses.NoHolder, bonuses.KindBomb, 1)

	assert.Equal(t, bonuses.Held{}, storage.Held(bonuses.NoHolder))
	require.NoError(t, storage.Flush(t.Context()))
	assert.Zero(t, persistence.Saves(), "nothing changed, so nothing is written")
}

func TestEachSpendTakesItsOwnKind(t *testing.T) {
	storage, _ := fresh(t)
	storage.Grant(alice, bonuses.KindBomb, 1)
	storage.Grant(alice, bonuses.KindEncloseClicks, 1)
	storage.Grant(alice, bonuses.KindSpreadClicks, 4)
	storage.Grant(alice, bonuses.KindRefill, 1)

	require.True(t, storage.SpendSpreadClick(alice))
	require.True(t, storage.SpendEnclose(alice))
	require.True(t, storage.SpendRefill(alice))
	require.False(t, storage.SpendRefill(alice))

	assert.Equal(t, bonuses.Held{Bomb: true, SpreadClicks: 3}, storage.Held(alice))
	assert.Equal(t, 25, storage.EnclosureMaxTiles())
	assert.Equal(t, 8, storage.SpreadClicks())
	assert.Equal(t, 3, storage.Enclosures())
}

func TestChargesSurviveARestart(t *testing.T) {
	storage, persistence := fresh(t)
	storage.Grant(alice, bonuses.KindBomb, 1)
	storage.Grant(bob, bonuses.KindSpreadClicks, 4)
	require.True(t, storage.SpendSpreadClick(bob))
	require.NoError(t, storage.Flush(t.Context()))

	after := loaded(t, persistence)

	assert.Equal(t, bonuses.Held{Bomb: true}, after.Held(alice))
	assert.Equal(t, bonuses.Held{SpreadClicks: 3}, after.Held(bob))
}

func TestASpentHandIsDeletedRatherThanKept(t *testing.T) {
	storage, persistence := fresh(t)
	storage.Grant(alice, bonuses.KindBomb, 1)
	require.NoError(t, storage.Flush(t.Context()))
	require.Contains(t, persistence.Stored(), alice)

	require.True(t, storage.SpendBomb(alice))
	require.NoError(t, storage.Flush(t.Context()))

	assert.NotContains(t, persistence.Stored(), alice)
}

func TestAFlushWithNothingChangedWritesNothing(t *testing.T) {
	storage, persistence := fresh(t)
	storage.Grant(alice, bonuses.KindBomb, 1)
	require.NoError(t, storage.Flush(t.Context()))

	require.NoError(t, storage.Flush(t.Context()))

	assert.Equal(t, 1, persistence.Saves())
}

func TestAFailedFlushIsRetriedWithTheLatestHands(t *testing.T) {
	storage, persistence := fresh(t)
	storage.Grant(alice, bonuses.KindSpreadClicks, 4)
	persistence.FailWith(errors.New("connection refused"))

	require.Error(t, storage.Flush(t.Context()))
	require.True(t, storage.SpendSpreadClick(alice))

	persistence.Heal()
	require.NoError(t, storage.Flush(t.Context()))

	assert.Equal(t, bonuses.Held{SpreadClicks: 3}, loaded(t, persistence).Held(alice))
}

func TestAFailedLoadRefusesTheBoot(t *testing.T) {
	persistence := inmemory_charge_storage.NewMemoryPersistence()
	persistence.FailWith(errors.New("connection refused"))

	storage := inmemory_charge_storage.New(inmemory_charge_storage.Config{}, rules, persistence, slog.New(slog.DiscardHandler))

	require.ErrorContains(t, storage.Load(t.Context()), "connection refused")
}
