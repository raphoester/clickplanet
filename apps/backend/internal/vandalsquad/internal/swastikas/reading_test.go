package swastikas_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/swastikas"
)

// Real drawings of the map: the swastikas players painted in season 0 and on 2026-11-06 (and two of them drawn again
// along the grid's other lines, elsewhere on the globe), and drawings that are none: the four the operator judged, the
// two the rules let through before they were tightened, text, crosses, circles with a cross, and drawings picked at
// random. Each tile is as planet's GetTiles answers it, with only its neighbours in the drawing.
type sample struct {
	Name       string       `json:"name"`
	Why        string       `json:"why"`
	IDs        []uint32     `json:"ids"`
	Points     [][3]float64 `json:"points"`
	Neighbours [][]uint32   `json:"neighbours"`
}

func samples(t *testing.T) (swastikasDrawn, others []sample) {
	t.Helper()
	raw, err := os.ReadFile("testdata/drawings.json")
	require.NoError(t, err)
	var file struct {
		Swastikas []sample `json:"swastikas"`
		Others    []sample `json:"others"`
	}
	require.NoError(t, json.Unmarshal(raw, &file))
	require.NotEmpty(t, file.Swastikas)
	require.NotEmpty(t, file.Others)
	return file.Swastikas, file.Others
}

func (s sample) drawing() drawings.Drawing {
	tiles := make([]drawings.Tile, len(s.IDs))
	for i, id := range s.IDs {
		neighbours := make([]drawings.TileID, len(s.Neighbours[i]))
		for j, neighbour := range s.Neighbours[i] {
			neighbours[j] = drawings.TileID(neighbour)
		}
		tiles[i] = drawings.Tile{
			ID:         drawings.TileID(id),
			Point:      drawings.Point{X: s.Points[i][0], Y: s.Points[i][1], Z: s.Points[i][2]},
			Neighbours: neighbours,
		}
	}
	return drawings.NewDrawing(tiles)
}

func TestEverySwastikaPaintedOnTheMapReadsAsOne(t *testing.T) {
	drawn, _ := samples(t)
	for _, s := range drawn {
		_, ok := swastikas.ReadingOf(s.drawing())
		assert.True(t, ok, "%s: %s", s.Name, s.Why)
	}
}

func TestNoOtherDrawingReadsAsOne(t *testing.T) {
	_, others := samples(t)
	for _, s := range others {
		reading, ok := swastikas.ReadingOf(s.drawing())
		assert.False(t, ok, "%s: %s, read from tile %d", s.Name, s.Why, reading.Centre)
	}
}

func TestAReadingNamesItsCentreAndTheShareOnItsBranches(t *testing.T) {
	drawn, _ := samples(t)

	reading, ok := swastikas.ReadingOf(drawn[0].drawing())

	require.True(t, ok)
	assert.Contains(t, drawn[0].IDs, uint32(reading.Centre))
	assert.InDelta(t, 0.9, reading.On, 0.1)
}

func TestTwoSwastikasSideBySideAreNotOneDrawing(t *testing.T) {
	drawn, _ := samples(t)
	both := append(drawn[0].drawing().Tiles(), drawn[1].drawing().Tiles()...)

	_, ok := swastikas.ReadingOf(drawings.NewDrawing(both))

	assert.False(t, ok, "the drawings are split before they are read: two apart are no one swastika")
}

func TestADrawingTooSmallOrTooBigIsNone(t *testing.T) {
	drawn, _ := samples(t)
	tiles := drawn[0].drawing().Tiles()

	_, ok := swastikas.ReadingOf(drawings.NewDrawing(tiles[:11]))
	assert.False(t, ok)

	big := make([]drawings.Tile, 0, 151)
	for i := range 151 {
		big = append(big, drawings.Tile{ID: drawings.TileID(i + 1)})
	}
	_, ok = swastikas.ReadingOf(drawings.NewDrawing(big))
	assert.False(t, ok)
}
