package clicks_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

const spacing = 0.004

func at(latitude, longitude float64) []float32 {
	lat, lon := latitude*math.Pi/180, longitude*math.Pi/180

	return []float32{
		float32(-math.Cos(lat) * math.Cos(lon)),
		float32(math.Sin(lat)),
		float32(math.Cos(lat) * math.Sin(lon)),
	}
}

func gridOf(t *testing.T, tiles ...[]float32) *clicks.Grid {
	t.Helper()

	var positions []float32
	for _, tile := range tiles {
		positions = append(positions, tile...)
	}
	geography, err := clicks.NewGeography(positions, nil)
	require.NoError(t, err)

	return clicks.NewGrid(geography, spacing)
}

func cell(row, column int) int {
	return row*clicks.GridColumns + column
}

func landCells(land []byte) []int {
	var cells []int
	for i, tiles := range land {
		if tiles > 0 {
			cells = append(cells, i)
		}
	}
	return cells
}

func TestATileInTheMiddleOfACellIsDrawnInItAlone(t *testing.T) {
	grid := gridOf(t, at(45.5, 10.5))

	assert.Equal(t, []int{cell(44, 190)}, landCells(grid.Land()), "row 44 is 45° to 46° north, column 190 is 10° to 11° east")
}

func TestTwoTilesInOneCellAreCountedTwice(t *testing.T) {
	grid := gridOf(t, at(-30.3, -60.3), at(-30.6, -60.6))

	assert.Equal(t, byte(2), grid.Land()[cell(120, 119)])
}

func TestATileOnTheAntimeridianIsDrawnOnBothEdges(t *testing.T) {
	grid := gridOf(t, at(0.5, 179.95))

	assert.Equal(t, []int{cell(89, 0), cell(89, 359)}, landCells(grid.Land()))
}

func TestATileByThePoleCoversTheLongitudesItStretchesOver(t *testing.T) {
	grid := gridOf(t, at(89.95, 0))

	cells := landCells(grid.Land())
	assert.Greater(t, len(cells), clicks.GridColumns/2, "at 89.95° north a tile is wider than half the world's longitudes")
	for _, drawn := range cells {
		assert.Less(t, drawn, clicks.GridColumns, "and it stays in the first row")
	}
}

func TestTheHeatIsHowOftenTheTilesOfEachCellChangedHands(t *testing.T) {
	grid := gridOf(t, at(-30.3, -60.3), at(-30.6, -60.6), at(45.5, 10.5))

	heat := grid.Heat(map[uint32]int{1: 3, 2: 1, 3: 2, 4: 7})

	assert.Equal(t, uint32(4), heat[cell(120, 119)])
	assert.Equal(t, uint32(2), heat[cell(44, 190)])

	var total uint32
	for _, changes := range heat {
		total += changes
	}
	assert.Equal(t, uint32(6), total, "a tile the map does not have is drawn nowhere")
}
