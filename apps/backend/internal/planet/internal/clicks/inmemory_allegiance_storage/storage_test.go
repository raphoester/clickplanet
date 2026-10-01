package inmemory_allegiance_storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ada   = clicks.AllegianceKey("ada")
	home  = clicks.AllegianceKey("home")
)

func TestATakeCountsInEveryTallyItNames(t *testing.T) {
	storage := New(cptime.NewFixedClock(epoch))

	storage.Record("fr", epoch, ada, home)

	assert.Equal(t, "fr", storage.Allegiance(ada).Flag())
	assert.Equal(t, "fr", storage.Allegiance(home).Flag())
}

func TestAnUnknownTallyHasNoFlag(t *testing.T) {
	assert.Empty(t, New(cptime.NewFixedClock(epoch)).Allegiance(ada).Flag())
}

func TestTalliesWithNoRecentTakeAreForgotten(t *testing.T) {
	clock := cptime.NewFixedClock(epoch)
	storage := New(clock)
	storage.Record("fr", clock.Now(), ada)

	clock.Advance(72 * time.Hour)
	storage.Record("es", clock.Now(), home)
	clock.Advance(time.Minute)
	storage.forgetFaded()

	assert.Empty(t, storage.Allegiance(ada).Flag())
	assert.Equal(t, "es", storage.Allegiance(home).Flag())
}
