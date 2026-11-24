package embedded_board_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/embedded_board"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpflags"
)

// Each region the game ships with, and two sold flags pinned in France, on whose ground each one sits.
func pinned() ([]clicks.Pinned, map[string]string) {
	pins := cpflags.Pins()
	flags := make([]clicks.Pinned, 0, len(pins)+2)
	for _, pin := range pins {
		flags = append(flags, clicks.Pinned{Flag: pin.Code, Pin: clicks.Pin{Latitude: pin.Latitude, Longitude: pin.Longitude}})
	}
	flags = append(flags,
		clicks.Pinned{Flag: "xb", Pin: clicks.Pin{Latitude: 48.1173, Longitude: -1.6778}},
		clicks.Pinned{Flag: "r-0000000001", Pin: clicks.Pin{Latitude: 41.9192, Longitude: 8.7386}})
	return flags, map[string]string{"gb-eng": "gb", "gb-sct": "gb", "gb-wls": "gb", "xb": "fr", "r-0000000001": "fr"}
}

func TestEachBoardSeatsTheFlagsOnAPinNearItOnTheirLandApartFromEveryCapital(t *testing.T) {
	flags, grounds := pinned()
	for _, name := range []clicks.BoardName{"earth-1", "earth-2", "siege-1"} {
		t.Run(string(name), func(t *testing.T) {
			ground, err := embedded_board.Shipped(name)
			require.NoError(t, err)

			seats := ground.Seats(flags)

			require.Len(t, seats, len(flags))
			for _, seat := range seats {
				pin := flags[slices.IndexFunc(flags, func(flag clicks.Pinned) bool { return flag.Flag == seat.Country })].Pin
				at, _ := ground.Geography().Position(seat.Tile)
				assert.Less(t, kilometres(at, pin.Latitude, pin.Longitude), 3*ground.Spacing()*6371, seat.Country)
				assert.Equal(t, grounds[seat.Country], ground.Borders().CountryOf(seat.Tile), seat.Country)
				assert.False(t, ground.Capitals().Is(seat.Tile), seat.Country)
				for _, next := range ground.Geography().Neighbours(seat.Tile) {
					assert.False(t, ground.Capitals().Is(next), "%s touches a capital", seat.Country)
				}
			}
		})
	}
}
