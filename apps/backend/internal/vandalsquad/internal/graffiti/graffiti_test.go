package graffiti_test

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
)

var (
	at  = time.Date(2026, 11, 6, 20, 15, 0, 0, time.UTC)
	ana = cpsession.AccountID(uuid.MustParse("01900000-0000-7000-8000-000000000001"))
	bob = cpsession.AccountID(uuid.MustParse("01900000-0000-7000-8000-000000000002"))
)

func drawing(tiles ...drawings.TileID) drawings.Drawing {
	made := make([]drawings.Tile, len(tiles))
	for i, id := range tiles {
		made[i] = drawings.Tile{ID: id, Point: point(-4, 22)}
	}
	return drawings.NewDrawing(made)
}

func sighting(account cpsession.AccountID, flag string, seen time.Time, tiles ...drawings.TileID) graffiti.Sighting {
	return graffiti.Sighting{Account: account, Flag: flag, Kind: graffiti.Swastika, Drawing: drawing(tiles...), At: seen}
}

func patrol() graffiti.Patrol {
	return graffiti.NewPatrol(time.Hour, "https://clickplanet.lol/play", &graffiti.SequentialIDs{})
}

const link = "https://clickplanet.lol/play?lat=-4.00&lon=22.00&zoom=20.0"

func TestANewDrawingIsFlaggedWhenItIsSeen(t *testing.T) {
	noted, fresh, err := patrol().Note(nil, sighting(ana, "ro", at, 3, 1, 2))

	require.NoError(t, err)
	assert.True(t, fresh)
	assert.Equal(t, graffiti.Snapshot{
		ID: graffiti.ID{15: 1}, Account: ana, Flag: "ro", Kind: graffiti.Swastika, Tiles: []drawings.TileID{1, 2, 3},
		FlaggedAt: at, SeenAt: at, Link: link,
	}, noted.TakeSnapshot())
}

func TestADrawingSeenAgainKeepsItsFirstFlaggingAndTakesItsNewShape(t *testing.T) {
	watch := patrol()
	first, _, err := watch.Note(nil, sighting(ana, "ro", at, 1, 2, 3))
	require.NoError(t, err)

	again, fresh, err := watch.Note([]graffiti.Graffiti{first}, sighting(ana, "ro", at.Add(time.Minute), 3, 4))

	require.NoError(t, err)
	assert.False(t, fresh)
	assert.Equal(t, graffiti.Snapshot{
		ID: graffiti.ID{15: 1}, Account: ana, Flag: "ro", Kind: graffiti.Swastika, Tiles: []drawings.TileID{3, 4},
		FlaggedAt: at, SeenAt: at.Add(time.Minute), Link: link,
	}, again.TakeSnapshot())
}

func TestAnotherPlayerAnotherFlagOtherTilesOrAForgottenPainterAreAnotherDrawing(t *testing.T) {
	watch := patrol()
	first, _, err := watch.Note(nil, sighting(ana, "ro", at, 1, 2))
	require.NoError(t, err)

	for name, other := range map[string]graffiti.Sighting{
		"another player": sighting(bob, "ro", at, 1, 2),
		"another flag":   sighting(ana, "br", at, 1, 2),
		"other tiles":    sighting(ana, "ro", at, 7, 8),
	} {
		_, fresh, err := watch.Note([]graffiti.Graffiti{first}, other)
		require.NoError(t, err)
		assert.True(t, fresh, name)
	}

	_, fresh, err := watch.Note([]graffiti.Graffiti{first.Anonymized()}, sighting(ana, "ro", at, 1, 2))
	require.NoError(t, err)
	assert.True(t, fresh, "a forgotten painter's graffiti is nobody's")
}

func TestAnAnonymizedGraffitiKeepsAllButItsPainter(t *testing.T) {
	first, _, err := patrol().Note(nil, sighting(ana, "ro", at, 1, 2))
	require.NoError(t, err)

	want := first.TakeSnapshot()
	want.Account = cpsession.NoAccount
	assert.Equal(t, []graffiti.Snapshot{want}, snapshots(graffiti.AllAnonymized([]graffiti.Graffiti{first})))
}

func snapshots(all []graffiti.Graffiti) []graffiti.Snapshot {
	taken := make([]graffiti.Snapshot, len(all))
	for i, one := range all {
		taken[i] = one.TakeSnapshot()
	}
	return taken
}

func TestASnapshotMissingAPartDoesNotRestore(t *testing.T) {
	valid := graffiti.Snapshot{
		ID: graffiti.ID{15: 1}, Flag: "ro", Kind: graffiti.Swastika, Tiles: []drawings.TileID{1},
		FlaggedAt: at, SeenAt: at, Link: link,
	}
	_, err := valid.Restore()
	require.NoError(t, err, "a graffiti whose painter was forgotten restores")

	for name, change := range map[string]func(*graffiti.Snapshot){
		"no id":       func(s *graffiti.Snapshot) { s.ID = graffiti.ID{} },
		"no flag":     func(s *graffiti.Snapshot) { s.Flag = "" },
		"no kind":     func(s *graffiti.Snapshot) { s.Kind = "smiley" },
		"no tile":     func(s *graffiti.Snapshot) { s.Tiles = nil },
		"never":       func(s *graffiti.Snapshot) { s.FlaggedAt = time.Time{} },
		"seen before": func(s *graffiti.Snapshot) { s.SeenAt = at.Add(-time.Minute) },
		"no link":     func(s *graffiti.Snapshot) { s.Link = "" },
	} {
		broken := valid
		change(&broken)
		_, err := broken.Restore()
		assert.ErrorIs(t, err, graffiti.ErrInvalidGraffiti, name)
	}
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
