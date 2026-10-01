package inmemory_allegiance_storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func TestAClickCountsForItsAccountAndItsScope(t *testing.T) {
	storage := New(cptime.NewFixedClock(epoch))

	storage.Record(clicks.Payer{Scope: "2001:db8::/64", Account: "acc-fr"}, "fr")

	assert.Equal(t, "fr", storage.OfAccount("acc-fr").Flag())
	assert.Equal(t, "fr", storage.OfScope("2001:db8::/64").Flag())
}

func TestAClickWithNoAccountCountsForItsScopeAlone(t *testing.T) {
	storage := New(cptime.NewFixedClock(epoch))

	storage.Record(clicks.Payer{Scope: "2001:db8::/64"}, "fr")

	assert.Equal(t, "fr", storage.OfScope("2001:db8::/64").Flag())
	assert.Empty(t, storage.accounts)
}

func TestAnUnknownAccountHasNoFlag(t *testing.T) {
	assert.Empty(t, New(cptime.NewFixedClock(epoch)).OfAccount("nobody").Flag())
}

func TestAllegiancesWithNoRecentClickAreForgotten(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	storage := New(clock)
	storage.Record(clicks.Payer{Scope: "2001:db8::/64", Account: "acc-old"}, "fr")

	clock.Advance(72 * time.Hour)
	storage.Record(clicks.Payer{Scope: "2001:db8:1::/64", Account: "acc-new"}, "es")
	clock.Advance(time.Minute)
	storage.forgetFaded()

	assert.Empty(t, storage.OfAccount("acc-old").Flag())
	assert.Empty(t, storage.OfScope("2001:db8::/64").Flag())
	assert.Equal(t, "es", storage.OfAccount("acc-new").Flag())
}
