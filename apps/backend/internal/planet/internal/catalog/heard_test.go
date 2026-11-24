package catalog_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var (
	rennes  = clicks.Pin{Latitude: 48.1173, Longitude: -1.6778}
	quimper = clicks.Pin{Latitude: 47.996, Longitude: -4.102}
)

func TestAFlagNeverHeardKeepsNoPin(t *testing.T) {
	assert.Equal(t, catalog.HeardSnapshot{}, catalog.Heard{}.Moved(rennes, opened).TakeSnapshot())
}

func TestTheLatestPinHolds(t *testing.T) {
	heard := catalog.Heard{}.Opened(opened).Moved(rennes, opened).Moved(quimper, closed)

	assert.Equal(t, quimper, heard.TakeSnapshot().Pin)
	assert.Equal(t, quimper, heard.Moved(rennes, opened).TakeSnapshot().Pin, "an older pin heard after it changes nothing")
}

func TestAPinToldAgainAtItsTimePutsItRight(t *testing.T) {
	heard := catalog.Heard{}.Opened(opened).Moved(rennes, opened)

	assert.Equal(t, quimper, heard.Moved(quimper, opened).TakeSnapshot().Pin, "the restatement at boot")
}

func TestAFlagClosedOrOpenedAgainKeepsItsPin(t *testing.T) {
	heard := catalog.Heard{}.Opened(opened).Moved(rennes, opened).Closed(closed).Opened(closed.Add(time.Hour))

	assert.Equal(t, catalog.HeardSnapshot{State: catalog.Open, DecidedAt: closed.Add(time.Hour), Pin: rennes, PinnedAt: opened},
		heard.TakeSnapshot())
}

func TestTheFlagsOnAPinAreEveryFlagOpenWithAPin(t *testing.T) {
	flags := catalog.NewCatalog(catalog.NewMemoryStore(), catalog.Unlocked)
	for flag, pin := range map[string]clicks.Pin{"r-0000000002": quimper, brittanyCity: rennes, "r-0000000003": rennes} {
		require.NoError(t, flags.Open(t.Context(), flag, opened))
		require.NoError(t, flags.Move(t.Context(), flag, pin, opened))
	}
	require.NoError(t, flags.Close(t.Context(), "r-0000000003", closed))
	require.NoError(t, flags.Open(t.Context(), "r-0000000004", opened))

	pinned, err := flags.Pinned(t.Context())

	require.NoError(t, err)
	assert.Equal(t, []clicks.Pinned{
		{Flag: "r-0000000002", Pin: quimper},
		{Flag: brittanyCity, Pin: rennes},
	}, pinned, "a closed flag sits nowhere, nor one with no pin heard yet")
}

func TestAFlagClosedUnderALockSitsNowhere(t *testing.T) {
	flags := catalog.NewCatalog(catalog.NewMemoryStore(), func() bool { return true })
	require.NoError(t, flags.Open(t.Context(), brittanyCity, opened))
	require.NoError(t, flags.Move(t.Context(), brittanyCity, rennes, opened))
	require.NoError(t, flags.Close(t.Context(), brittanyCity, closed))

	pinned, err := flags.Pinned(t.Context())

	require.NoError(t, err)
	assert.NotContains(t, pinned, clicks.Pinned{Flag: brittanyCity, Pin: rennes})
}
