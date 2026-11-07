package graffiti_test

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
)

var (
	at  = time.Date(2026, 11, 6, 20, 15, 0, 0, time.UTC)
	ana = cpsession.AccountID(uuid.MustParse("01900000-0000-7000-8000-000000000001"))
	bob = cpsession.AccountID(uuid.MustParse("01900000-0000-7000-8000-000000000002"))
)

func swastika(account cpsession.AccountID, flag string, seen time.Time, tiles ...drawings.TileID) graffiti.Graffiti {
	return graffiti.Graffiti{Account: account, Flag: flag, Kind: graffiti.Swastika, Tiles: tiles, FlaggedAt: seen, SeenAt: seen}
}

func TestADrawingSeenAgainKeepsItsFirstFlaggingAndTakesItsNewShape(t *testing.T) {
	book := graffiti.NewBook(10)
	first := swastika(ana, "ro", at, 1, 2, 3)
	assert.False(t, book.Flagged(first))
	book.Keep(first)

	again := swastika(ana, "ro", at.Add(time.Minute), 3, 4)
	assert.True(t, book.Flagged(again))
	book.Keep(again)

	want := swastika(ana, "ro", at.Add(time.Minute), 3, 4)
	want.FlaggedAt = at
	assert.Equal(t, []graffiti.Graffiti{want}, book.Entries())
}

func TestAnotherPlayerAnotherFlagOrOtherTilesAreAnotherDrawing(t *testing.T) {
	book := graffiti.NewBook(10)
	book.Keep(swastika(ana, "ro", at, 1, 2))

	for _, other := range []graffiti.Graffiti{
		swastika(bob, "ro", at, 1, 2), swastika(ana, "br", at, 1, 2), swastika(ana, "ro", at, 7, 8),
	} {
		assert.False(t, book.Flagged(other))
	}
}

func TestTheNewestComesFirstAndTheOldestGoesOnceTheBookIsFull(t *testing.T) {
	book := graffiti.NewBook(2)
	book.Keep(swastika(ana, "ro", at, 1))
	book.Keep(swastika(ana, "ro", at, 2))
	book.Keep(swastika(ana, "ro", at, 3))

	assert.Equal(t, []graffiti.Graffiti{swastika(ana, "ro", at, 3), swastika(ana, "ro", at, 2)}, book.Entries())
}

// Longitude 0 is on -x, and the east is towards +z: the frontend's globe.
func point(lat, lon float64) drawings.Point {
	la, lo := lat*math.Pi/180, lon*math.Pi/180
	return drawings.Point{X: -math.Cos(la) * math.Cos(lo), Y: math.Sin(la), Z: math.Cos(la) * math.Sin(lo)}
}

func TestTheLinkOpensTheGlobeOnTheDrawingsMiddle(t *testing.T) {
	link := graffiti.LinkOf("https://clickplanet.lol/play", []drawings.Point{point(-4, 22), point(-4.2, 22.4)})

	assert.Equal(t, "https://clickplanet.lol/play?lat=-4.10&lon=22.20&zoom=20.0", link, "a drawing is framed at the closest zoom")
}

func TestABigDrawingIsFramedFromFarther(t *testing.T) {
	link := graffiti.LinkOf("https://clickplanet.lol/play", []drawings.Point{point(0, -10), point(0, 10)})

	assert.Equal(t, "https://clickplanet.lol/play?lat=0.00&lon=0.00&zoom=2.3", link)
}
