package clicks

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var epoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func clicksFor(allegiance Allegiance, country string, times int, at time.Time) Allegiance {
	for range times {
		allegiance = allegiance.With(country, at)
	}

	return allegiance
}

func TestNoClickIsNoFlag(t *testing.T) {
	assert.Empty(t, Allegiance{}.Flag())
}

func TestTheMainFlagIsTheOneClickedForMost(t *testing.T) {
	allegiance := clicksFor(Allegiance{}, "pl", 20, epoch)
	allegiance = allegiance.With("ad", epoch)

	assert.Equal(t, "pl", allegiance.Flag(), "one click for another flag does not change it")
}

func TestAnOldFlagFadesBehindANewOne(t *testing.T) {
	loyal := clicksFor(Allegiance{}, "pl", 10, epoch)

	assert.Equal(t, "pl", clicksFor(loyal, "ad", 3, epoch).Flag())
	assert.Equal(t, "ad", clicksFor(loyal, "ad", 3, epoch.Add(2*flagHalfLife)).Flag(),
		"ten clicks two half-lives ago weigh 2.5")
}

func TestATieGoesToTheCodeThatSortsFirst(t *testing.T) {
	allegiance := Allegiance{}.With("pl", epoch).With("ad", epoch)

	assert.Equal(t, "ad", allegiance.Flag())
}

func TestWithLeavesTheAllegianceItWasBuiltFrom(t *testing.T) {
	before := Allegiance{}.With("pl", epoch)

	after := clicksFor(before, "ad", 5, epoch)

	assert.Equal(t, "pl", before.Flag())
	assert.Equal(t, "ad", after.Flag())
}

func TestAFadedFlagIsDroppedFromTheTally(t *testing.T) {
	allegiance := Allegiance{}.With("pl", epoch).With("ad", epoch.Add(7*flagHalfLife))

	assert.Equal(t, map[string]float64{"ad": 1}, allegiance.weights)
}

func TestAnAllegianceWithNoClickInItsMemoryHasFaded(t *testing.T) {
	allegiance := Allegiance{}.With("pl", epoch)

	assert.False(t, allegiance.Faded(epoch.Add(flagMemory)))
	assert.True(t, allegiance.Faded(epoch.Add(flagMemory+time.Second)))
}
