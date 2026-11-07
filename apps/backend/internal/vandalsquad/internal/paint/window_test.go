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

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

var (
	noon = time.Date(2026, 11, 6, 20, 0, 0, 0, time.UTC)
	ana  = cpsession.AccountID(uuid.MustParse("01900000-0000-7000-8000-000000000001"))
	bob  = cpsession.AccountID(uuid.MustParse("01900000-0000-7000-8000-000000000002"))
)

func TestEachPlayersTilesAreItsPaintingOfEachFlag(t *testing.T) {
	window := paint.NewWindow(time.Hour, &clock{now: noon})
	for _, take := range []paint.Take{
		{Account: ana, Tile: 3, Flag: "ro", At: noon.Add(-time.Minute)},
		{Account: ana, Tile: 1, Flag: "ro", At: noon.Add(-time.Minute)},
		{Account: ana, Tile: 2, Flag: "ro", At: noon.Add(-time.Minute)},
		{Account: ana, Tile: 9, Flag: "fr", At: noon.Add(-time.Minute)},
		{Account: bob, Tile: 4, Flag: "cz", At: noon.Add(-time.Minute)},
	} {
		window.Keep(take)
	}

	assert.Equal(t, []paint.Painting{
		{Account: ana, Flag: "fr", Tiles: []drawings.TileID{9}},
		{Account: ana, Flag: "ro", Tiles: []drawings.TileID{1, 2, 3}},
		{Account: bob, Flag: "cz", Tiles: []drawings.TileID{4}},
	}, window.Paintings(1))
	assert.Equal(t, []paint.Painting{{Account: ana, Flag: "ro", Tiles: []drawings.TileID{1, 2, 3}}}, window.Paintings(3))
}

func TestATakeTwiceOrLateKeepsThePlayersLatestOfTheTile(t *testing.T) {
	window := paint.NewWindow(time.Hour, &clock{now: noon})

	window.Keep(paint.Take{Account: ana, Tile: 1, Flag: "fr", At: noon.Add(-time.Minute)})
	window.Keep(paint.Take{Account: ana, Tile: 1, Flag: "ro", At: noon.Add(-2 * time.Minute)})
	window.Keep(paint.Take{Account: ana, Tile: 1, Flag: "fr", At: noon.Add(-time.Minute)})

	assert.Equal(t, []paint.Painting{{Account: ana, Flag: "fr", Tiles: []drawings.TileID{1}}}, window.Paintings(1))
}

func TestATakeOlderThanTheSpanOrOfNobodyIsNotKept(t *testing.T) {
	window := paint.NewWindow(time.Hour, &clock{now: noon})

	window.Keep(paint.Take{Account: ana, Tile: 1, Flag: "fr", At: noon.Add(-2 * time.Hour)})
	window.Keep(paint.Take{Account: cpsession.NoAccount, Tile: 2, Flag: "fr", At: noon})

	assert.Empty(t, window.Paintings(1))
}

func TestTheSpanMovesOnAndForgettingDropsWhatLeftIt(t *testing.T) {
	now := &clock{now: noon}
	window := paint.NewWindow(time.Hour, now)
	window.Keep(paint.Take{Account: ana, Tile: 1, Flag: "fr", At: noon.Add(-50 * time.Minute)})
	window.Keep(paint.Take{Account: ana, Tile: 2, Flag: "fr", At: noon})

	now.now = noon.Add(20 * time.Minute)
	assert.Equal(t, []paint.Painting{{Account: ana, Flag: "fr", Tiles: []drawings.TileID{2}}}, window.Paintings(1),
		"a take that left the span is not painted any more, forgotten or not")

	window.Forget()
	now.now = noon.Add(2 * time.Hour)
	window.Forget()
	assert.Empty(t, window.Paintings(1))
}
