package clicks_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Ten tiles a degree apart on the equator, from longitude 0, each touching the next: 1 to 8 are a country's land, its
// capital on 2, and 9 and 10 are sea.
func equator() *clicks.Ground {
	points := make([]clicks.Pin, 10)
	edges := make([]clicks.Edge, 0, 9)
	for i := range points {
		points[i] = clicks.Pin{Longitude: float64(i)}
		if i > 0 {
			edges = append(edges, clicks.Edge{From: uint32(i), To: uint32(i + 1)}) //nolint:gosec // ten tiles.
		}
	}
	return clicks.PlacedGround(clicks.BordersOf(10, []uint32{1, 2, 3, 4, 5, 6, 7, 8}),
		clicks.NewCapitals(map[string]uint32{"l1": 2}), points, edges)
}

func at(flag string, longitude float64) clicks.Pinned {
	return clicks.Pinned{Flag: flag, Pin: clicks.Pin{Longitude: longitude}}
}

func TestAPinIsAPointAsTheTilePositionsPlaceIt(t *testing.T) {
	assert.InDelta(t, -1, clicks.Pin{}.Point().X, 1e-9, "longitude 0 on -x")
	assert.InDelta(t, 1, clicks.Pin{Latitude: 90}.Point().Y, 1e-9, "the north pole on y")
	assert.InDelta(t, 1, clicks.Pin{Longitude: 90}.Point().Z, 1e-9)
}

func TestAFlagSitsOnTheLandTileNearestItsPin(t *testing.T) {
	assert.Equal(t, []clicks.Capital{{Country: "r-0000000001", Tile: 6}}, equator().Seats([]clicks.Pinned{at("r-0000000001", 5.1)}))
}

func TestAPinOnOrNextToACapitalSitsOnTheNextFreeLand(t *testing.T) {
	ground := equator()

	assert.Equal(t, []clicks.Capital{{Country: "gb-eng", Tile: 4}}, ground.Seats([]clicks.Pinned{at("gb-eng", 1)}),
		"2 is the capital, 1 and 3 touch it")
	assert.Equal(t, []clicks.Capital{{Country: "gb-eng", Tile: 4}}, ground.Seats([]clicks.Pinned{at("gb-eng", 2.2)}))
}

func TestAPinAtSeaSitsOnTheNearestLand(t *testing.T) {
	assert.Equal(t, []clicks.Capital{{Country: "r-0000000001", Tile: 8}}, equator().Seats([]clicks.Pinned{at("r-0000000001", 9)}))
}

func TestTwoFlagsOnOnePlaceSitApartInTheOrderOfTheirCodes(t *testing.T) {
	seats := equator().Seats([]clicks.Pinned{at("r-0000000002", 5.2), at("r-0000000001", 5.2)})

	assert.Equal(t, []clicks.Capital{{Country: "r-0000000001", Tile: 6}, {Country: "r-0000000002", Tile: 8}}, seats,
		"5 and 7 touch the first seat")
}

func TestAFlagPinnedTwiceSitsOnce(t *testing.T) {
	assert.Len(t, equator().Seats([]clicks.Pinned{at("r-0000000001", 5), at("r-0000000001", 6)}), 1)
}

func TestAFlagWithACapitalOfItsOwnGetsNoSeat(t *testing.T) {
	assert.Empty(t, equator().Seats([]clicks.Pinned{at("l1", 6)}))
}

func TestABoardWithNoLandFreeSeatsNobody(t *testing.T) {
	sea := clicks.PlacedGround(clicks.BordersOf(2), clicks.NewCapitals(map[string]uint32{}),
		[]clicks.Pin{{}, {Longitude: 1}}, []clicks.Edge{{From: 1, To: 2}})

	assert.Empty(t, sea.Seats([]clicks.Pinned{at("r-0000000001", 0)}))
}

func TestTheMapPlaysItsSeatsAsCapitalsUntilAnotherGroundPlays(t *testing.T) {
	ground := equator()
	played := clicks.PlayedOn(ground)

	played.Seat([]clicks.Capital{{Country: "r-0000000001", Tile: 6}})

	assert.Equal(t, []clicks.Capital{{Country: "l1", Tile: 2}, {Country: "r-0000000001", Tile: 6}}, played.Capitals().All())
	assert.Equal(t, []clicks.Capital{{Country: "l1", Tile: 2}}, ground.Capitals().All(), "the board keeps its own")
	played.Play(ground)
	assert.False(t, played.Capitals().Has("r-0000000001"))
}
