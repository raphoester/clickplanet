package inmemory_charge_storage_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/inmemory_charge_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)

const (
	alice bonuses.Holder = "alice"
	bob   bonuses.Holder = "bob"
)

var rules = bonuses.ChargesConfig{TTL: 24 * time.Hour, SpreadClicks: 8, EnclosureMaxTiles: 25}

func loaded(t *testing.T, persistence inmemory_charge_storage.Persistence, clock cptime.Clock) *inmemory_charge_storage.Storage {
	t.Helper()

	storage := inmemory_charge_storage.New(inmemory_charge_storage.Config{}, rules, persistence, clock,
		slog.New(slog.DiscardHandler))
	require.NoError(t, storage.Load(t.Context()))

	return storage
}

func fresh(t *testing.T) (*inmemory_charge_storage.Storage, *inmemory_charge_storage.MemoryPersistence, *cptime.FixedClock) {
	t.Helper()

	clock := cptime.NewFixedClock(epoch)
	persistence := inmemory_charge_storage.NewMemoryPersistence()

	return loaded(t, persistence, clock), persistence, clock
}

func TestAChargeBelongsToItsHolder(t *testing.T) {
	storage, _, _ := fresh(t)
	storage.Grant(alice, bonuses.KindBomb)

	assert.False(t, storage.SpendBomb(bob))
	assert.Equal(t, bonuses.Held{}, storage.Held(bob))
	assert.True(t, storage.SpendBomb(alice))
	assert.False(t, storage.SpendBomb(alice), "a bomb that went off is gone")
}

func TestACallerWithNoAccountHoldsNothing(t *testing.T) {
	storage, persistence, _ := fresh(t)

	storage.Grant(bonuses.NoHolder, bonuses.KindBomb)

	assert.Equal(t, bonuses.Held{}, storage.Held(bonuses.NoHolder))
	require.NoError(t, storage.Flush(t.Context()))
	assert.Zero(t, persistence.Saves(), "nothing changed, so nothing is written")
}

func TestEachSpendTakesItsOwnKind(t *testing.T) {
	storage, _, _ := fresh(t)
	storage.Grant(alice, bonuses.KindBomb)
	storage.Grant(alice, bonuses.KindEncloseClicks)
	storage.Grant(alice, bonuses.KindSpreadClicks)
	storage.Grant(alice, bonuses.KindRefill)

	require.True(t, storage.SpendSpreadClick(alice))
	require.True(t, storage.SpendEnclose(alice))
	require.True(t, storage.SpendRefill(alice))
	require.False(t, storage.SpendRefill(alice))

	assert.Equal(t, bonuses.Held{Bomb: true, SpreadClicks: 7}, storage.Held(alice))
	assert.Equal(t, 25, storage.EnclosureMaxTiles())
	assert.Equal(t, 8, storage.SpreadClicks())
}

func TestChargesSurviveARestart(t *testing.T) {
	storage, persistence, clock := fresh(t)
	storage.Grant(alice, bonuses.KindBomb)
	storage.Grant(bob, bonuses.KindSpreadClicks)
	require.True(t, storage.SpendSpreadClick(bob))
	require.NoError(t, storage.Flush(t.Context()))

	after := loaded(t, persistence, clock)

	assert.Equal(t, bonuses.Held{Bomb: true}, after.Held(alice))
	assert.Equal(t, bonuses.Held{SpreadClicks: 7}, after.Held(bob))
}

func TestASpentHandIsDeletedRatherThanKept(t *testing.T) {
	storage, persistence, _ := fresh(t)
	storage.Grant(alice, bonuses.KindBomb)
	require.NoError(t, storage.Flush(t.Context()))
	require.Contains(t, persistence.Stored(), alice)

	require.True(t, storage.SpendBomb(alice))
	require.NoError(t, storage.Flush(t.Context()))

	assert.NotContains(t, persistence.Stored(), alice)
}

func TestALapsedHandIsDeletedOnTheNextFlush(t *testing.T) {
	storage, persistence, clock := fresh(t)
	storage.Grant(alice, bonuses.KindBomb)
	require.NoError(t, storage.Flush(t.Context()))

	clock.Advance(24 * time.Hour)
	require.NoError(t, storage.Flush(t.Context()))

	assert.Empty(t, persistence.Stored(), "a charge nobody used in a day is not kept forever")
}

func TestAFlushWithNothingChangedWritesNothing(t *testing.T) {
	storage, persistence, _ := fresh(t)
	storage.Grant(alice, bonuses.KindBomb)
	require.NoError(t, storage.Flush(t.Context()))

	require.NoError(t, storage.Flush(t.Context()))

	assert.Equal(t, 1, persistence.Saves())
}

func TestAFailedFlushIsRetriedWithTheLatestHands(t *testing.T) {
	storage, persistence, clock := fresh(t)
	storage.Grant(alice, bonuses.KindSpreadClicks)
	persistence.FailWith(errors.New("connection refused"))

	require.Error(t, storage.Flush(t.Context()))
	require.True(t, storage.SpendSpreadClick(alice))

	persistence.Heal()
	require.NoError(t, storage.Flush(t.Context()))

	assert.Equal(t, bonuses.Held{SpreadClicks: 7}, loaded(t, persistence, clock).Held(alice))
}

func TestAFailedLoadRefusesTheBoot(t *testing.T) {
	persistence := inmemory_charge_storage.NewMemoryPersistence()
	persistence.FailWith(errors.New("connection refused"))

	storage := inmemory_charge_storage.New(inmemory_charge_storage.Config{}, rules, persistence,
		cptime.NewFixedClock(epoch), slog.New(slog.DiscardHandler))

	require.ErrorContains(t, storage.Load(t.Context()), "connection refused")
}

func TestAHandThatLapsedWhileTheServerWasDownIsNotLoaded(t *testing.T) {
	storage, persistence, clock := fresh(t)
	storage.Grant(alice, bonuses.KindBomb)
	require.NoError(t, storage.Flush(t.Context()))

	clock.Advance(25 * time.Hour)
	after := loaded(t, persistence, clock)
	require.NoError(t, after.Flush(t.Context()))

	assert.Equal(t, bonuses.Held{}, after.Held(alice))
	assert.Empty(t, persistence.Stored(), "and its row goes with the first flush")
}
