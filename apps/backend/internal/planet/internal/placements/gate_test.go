package placements_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/allegiances"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/placements"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcountries"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type board struct {
	owners  map[uint32]string
	shields map[uint32]int
}

func (b board) Owner(tile uint32) (string, bool) {
	if tile > 100 {
		return "", false
	}
	return b.owners[tile], true
}

func (b board) Shields(tile uint32) int { return b.shields[tile] }

var twoATile = func() bonuses.Rules {
	rules := bonuses.TestRules()
	rules.Shield.PerTile = 2
	return rules
}

func gate(t *testing.T, pledged string) placements.Gate {
	t.Helper()

	return gateOn(t, allegiances.NewMemoryStore(), pledged)
}

func gateOn(t *testing.T, store *allegiances.MemoryStore, pledged string) placements.Gate {
	t.Helper()

	book := allegiances.NewBook(store, cpcountries.New(), func() allegiances.Lock { return false },
		cptime.NewFixedClock(takenAt), &allegiances.RecordedPledges{})
	if pledged != "" {
		require.NoError(t, book.Pledge(t.Context(), clicks.Payer{Account: string(alice)}, pledged))
	}
	tiles := board{owners: map[uint32]string{7: "fr", 8: "de", 9: "fr"}, shields: map[uint32]int{9: 2}}
	return placements.NewGate(tiles, cpcountries.New(), book, twoATile)
}

func aim(t *testing.T, g placements.Gate, tile uint32, country string) (placements.Aim, error) {
	t.Helper()

	return g.Aim(t.Context(), clicks.Payer{Account: string(alice)}, tile, country) //nolint:wrapcheck // the test reads the gate's own error.
}

func TestATileOfTheFlagIsAimedWithTheMostItMayHold(t *testing.T) {
	aimed, err := aim(t, gate(t, "fr"), 7, "fr")

	require.NoError(t, err)
	assert.Equal(t, placements.Aim{Tile: 7, Country: "fr", Most: 2}, aimed)
}

func TestTheGateRefusesWhatTheMapWouldRefuse(t *testing.T) {
	g := gate(t, "fr")

	_, err := aim(t, g, 7, "nowhere")
	require.ErrorIs(t, err, clicks.ErrUnknownCountry)
	_, err = aim(t, g, 101, "fr")
	require.ErrorIs(t, err, clicks.ErrTileOutOfRange)
	_, err = aim(t, g, 8, "fr")
	require.ErrorIs(t, err, clicks.ErrNotYourTile)
	_, err = aim(t, g, 10, "fr")
	require.ErrorIs(t, err, clicks.ErrNotYourTile, "an empty tile wears no flag")
	_, err = aim(t, g, 9, "fr")
	require.ErrorIs(t, err, clicks.ErrTileFull)
}

func TestTheGateRefusesAFlagTheCallerDoesNotPlayFor(t *testing.T) {
	_, err := aim(t, gate(t, "fr"), 8, "de")
	require.ErrorIs(t, err, allegiances.ErrOtherFlag)

	_, err = aim(t, gate(t, ""), 7, "fr")
	require.ErrorIs(t, err, allegiances.ErrNoFlag)
}

func TestAFlagThatCannotBeReadRefusesNoPlacement(t *testing.T) {
	store := allegiances.NewMemoryStore()
	g := gateOn(t, store, "fr")
	store.FailWith(errors.New("postgres is down"))

	aimed, err := aim(t, g, 7, "fr")

	require.NoError(t, err)
	assert.Equal(t, placements.Aim{Tile: 7, Country: "fr", Most: 2}, aimed)
}
