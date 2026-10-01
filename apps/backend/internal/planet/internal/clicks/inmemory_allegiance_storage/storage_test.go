package inmemory_allegiance_storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func TestATakeCountsForItsAccountAndItsScope(t *testing.T) {
	storage := New(cptime.NewFixedClock(epoch))

	storage.Record("acc-fr", "2001:db8::/64", "fr", epoch)

	assert.Equal(t, "fr", storage.OfAccount("acc-fr").Flag())
	assert.Equal(t, "fr", storage.OfScope("2001:db8::/64").Flag())
}

func TestATakeWithNoAccountCountsForItsScopeAlone(t *testing.T) {
	storage := New(cptime.NewFixedClock(epoch))

	storage.Record("", "2001:db8::/64", "fr", epoch)

	assert.Equal(t, "fr", storage.OfScope("2001:db8::/64").Flag())
	assert.Empty(t, storage.accounts)
}

func TestAnUnknownAccountHasNoFlag(t *testing.T) {
	assert.Empty(t, New(cptime.NewFixedClock(epoch)).OfAccount("nobody").Flag())
}

func TestAllegiancesWithNoRecentClickAreForgotten(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	storage := New(clock)
	storage.Record("acc-old", "2001:db8::/64", "fr", clock.Now())

	clock.Advance(72 * time.Hour)
	storage.Record("acc-new", "2001:db8:1::/64", "es", clock.Now())
	clock.Advance(time.Minute)
	storage.forgetFaded()

	assert.Empty(t, storage.OfAccount("acc-old").Flag())
	assert.Empty(t, storage.OfScope("2001:db8::/64").Flag())
	assert.Equal(t, "es", storage.OfAccount("acc-new").Flag())
}
