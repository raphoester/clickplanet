package paint_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/paint"
)

var (
	noon = time.Date(2026, 11, 6, 20, 0, 0, 0, time.UTC)
	ana  = cpsession.AccountID(uuid.MustParse("01900000-0000-7000-8000-000000000001"))
	bob  = cpsession.AccountID(uuid.MustParse("01900000-0000-7000-8000-000000000002"))
)

func TestEachPlayersTilesAreItsPaintingOfEachFlag(t *testing.T) {
	takes := []paint.Take{
		{Account: ana, Tile: 3, Flag: "ro", At: noon},
		{Account: ana, Tile: 1, Flag: "ro", At: noon},
		{Account: ana, Tile: 2, Flag: "ro", At: noon},
		{Account: ana, Tile: 9, Flag: "fr", At: noon},
		{Account: bob, Tile: 4, Flag: "cz", At: noon},
	}

	assert.Equal(t, []paint.Painting{
		{Account: ana, Flag: "fr", Tiles: []drawings.TileID{9}},
		{Account: ana, Flag: "ro", Tiles: []drawings.TileID{1, 2, 3}},
		{Account: bob, Flag: "cz", Tiles: []drawings.TileID{4}},
	}, paint.PaintingsOf(takes, 1))
	assert.Equal(t, []paint.Painting{{Account: ana, Flag: "ro", Tiles: []drawings.TileID{1, 2, 3}}}, paint.PaintingsOf(takes, 3))
}

func TestThePlayersLatestTakeOfATileIsTheOneThatCounts(t *testing.T) {
	takes := []paint.Take{
		{Account: ana, Tile: 1, Flag: "ro", At: noon},
		{Account: ana, Tile: 1, Flag: "fr", At: noon.Add(time.Minute)},
		{Account: ana, Tile: 1, Flag: "fr", At: noon.Add(time.Minute)},
	}

	assert.Equal(t, []paint.Painting{{Account: ana, Flag: "fr", Tiles: []drawings.TileID{1}}}, paint.PaintingsOf(takes, 1))
}

func TestATakeOfNobodyIsNoPainting(t *testing.T) {
	assert.Empty(t, paint.PaintingsOf([]paint.Take{{Account: cpsession.NoAccount, Tile: 2, Flag: "fr", At: noon}}, 1))
}

func TestAPaintingHoldsTheTilesItsPlayerStillHoldsForItsFlag(t *testing.T) {
	painting := paint.Painting{Account: ana, Flag: "ro", Tiles: []drawings.TileID{1, 2, 3, 4}}
	onMap := map[drawings.TileID]drawings.MapTile{
		1: {Tile: drawings.Tile{ID: 1}, Flag: "ro", Holder: ana},
		2: {Tile: drawings.Tile{ID: 2}, Flag: "ro", Holder: bob},
		3: {Tile: drawings.Tile{ID: 3}, Flag: "", Holder: cpsession.NoAccount},
	}

	assert.Equal(t, []drawings.Tile{{ID: 1}}, painting.Held(onMap),
		"another's take, a bomb's clear and a tile planet did not answer are not the player's drawing")
}
