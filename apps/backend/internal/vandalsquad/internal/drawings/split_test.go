package drawings_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
)

func TestTilesThatTouchAreOneDrawingAndTilesApartAreTwo(t *testing.T) {
	split := drawings.Split([]drawings.Tile{
		{ID: 5, Neighbours: []drawings.TileID{6}},
		{ID: 1, Neighbours: []drawings.TileID{2, 9}},
		{ID: 6, Neighbours: []drawings.TileID{5}},
		{ID: 2, Neighbours: []drawings.TileID{1, 3}},
		{ID: 3, Neighbours: []drawings.TileID{2}},
	})

	ids := make([][]drawings.TileID, len(split))
	for i, drawing := range split {
		ids[i] = drawing.IDs()
	}
	assert.Equal(t, [][]drawings.TileID{{1, 2, 3}, {5, 6}}, ids)
}

func TestTheRimIsEveryTileTouchedOutsideOnce(t *testing.T) {
	drawing := drawings.NewDrawing([]drawings.Tile{
		{ID: 1, Neighbours: []drawings.TileID{2, 9, 7}},
		{ID: 2, Neighbours: []drawings.TileID{1, 9, 8}},
	})

	assert.Equal(t, []drawings.TileID{7, 8, 9}, drawing.Rim())
}

func TestADrawingShowsWhenMostOfTheLandAroundItWearsAnotherFlag(t *testing.T) {
	assert.True(t, drawings.Visible("ro", []string{"cz", "cz", "cz", "ro"}), "a drawing on another flag's ground")
	assert.True(t, drawings.Visible("ro", []string{"", "", "cz"}), "on empty land")
	assert.True(t, drawings.Visible("ro", []string{"ro", "cz"}), "half and half")
	assert.False(t, drawings.Visible("br", []string{"br", "br", "br", "ro"}), "players painting over a drawing on their own ground")
	assert.True(t, drawings.Visible("br", nil), "an island")
}
