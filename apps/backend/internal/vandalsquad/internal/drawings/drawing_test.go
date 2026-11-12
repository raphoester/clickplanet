package drawings_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
)

// Longitude 0 is on -x, and the east is towards +z: the frontend's globe.
func at(lat, lon float64) drawings.Point {
	la, lo := lat*math.Pi/180, lon*math.Pi/180
	return drawings.Point{X: -math.Cos(la) * math.Cos(lo), Y: math.Sin(la), Z: math.Cos(la) * math.Sin(lo)}
}

func TestATileTwiceIsOneAndTheTilesAreInTheOrderOfTheirIDs(t *testing.T) {
	drawing := drawings.NewDrawing([]drawings.Tile{{ID: 9}, {ID: 4}, {ID: 9}})

	assert.Equal(t, 2, drawing.Size())
	assert.Equal(t, []drawings.TileID{4, 9}, drawing.IDs())
}

func TestALinkIsANeighbourInTheDrawingInTheMapsOrder(t *testing.T) {
	drawing := drawings.NewDrawing([]drawings.Tile{
		{ID: 1, Neighbours: []drawings.TileID{3, 7, 2}},
		{ID: 2, Neighbours: []drawings.TileID{1}},
		{ID: 3, Neighbours: []drawings.TileID{1, 8}},
	})

	assert.Equal(t, [][]int{{2, 1}, {0}, {0}}, drawing.Links(), "tiles 7 and 8 are not in it")
}

func TestTheGlobeIsSeenWithTheEastToTheRightAndTheNorthUp(t *testing.T) {
	for _, place := range []struct{ lat, lon float64 }{{0, 0}, {48, 2}, {-33, 151}, {64, -150}} {
		drawing := drawings.NewDrawing([]drawings.Tile{
			{ID: 1, Point: at(place.lat, place.lon)},
			{ID: 2, Point: at(place.lat, place.lon+0.2)},
			{ID: 3, Point: at(place.lat+0.2, place.lon)},
		})

		flats := drawing.Flats()

		assert.Greater(t, flats[1].X, flats[0].X, "%v: the tile east is to the right", place)
		assert.InDelta(t, flats[0].Y, flats[1].Y, 1e-3, "%v", place)
		assert.Greater(t, flats[2].Y, flats[0].Y, "%v: the tile north is up", place)
		assert.InDelta(t, flats[0].X, flats[2].X, 1e-3, "%v", place)
	}
}

func TestADrawingOnThePoleStillLiesFlat(t *testing.T) {
	drawing := drawings.NewDrawing([]drawings.Tile{
		{ID: 1, Point: drawings.Point{Y: 1}},
		{ID: 2, Point: at(89.9, 0)},
	})

	for _, flat := range drawing.Flats() {
		assert.False(t, math.IsNaN(flat.X) || math.IsNaN(flat.Y))
	}
}
