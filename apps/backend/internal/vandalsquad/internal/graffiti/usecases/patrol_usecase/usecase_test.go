package patrol_usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/inmemory_graffiti_storage"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/usecases/patrol_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/paint"
)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

var (
	now = time.Date(2026, 11, 6, 20, 19, 0, 0, time.UTC)
	foe = cpsession.AccountID(uuid.MustParse("01900000-0000-7000-8000-00000000000f"))
	cz  = cpsession.AccountID(uuid.MustParse("01900000-0000-7000-8000-0000000000c2"))
)

// A rim tile of a drawing's tile: off the drawing, one per tile.
const rimOffset = 1_000_000

// The first real swastika and the first other drawing of the detector's samples, each tile touching one rim tile.
func samples(t *testing.T) (swastika, other []drawings.Tile) {
	t.Helper()
	raw, err := os.ReadFile("../../../swastikas/testdata/drawings.json")
	require.NoError(t, err)
	var file struct {
		Swastikas []struct {
			IDs        []uint32     `json:"ids"`
			Points     [][3]float64 `json:"points"`
			Neighbours [][]uint32   `json:"neighbours"`
		} `json:"swastikas"`
		Others []struct {
			IDs        []uint32     `json:"ids"`
			Points     [][3]float64 `json:"points"`
			Neighbours [][]uint32   `json:"neighbours"`
		} `json:"others"`
	}
	require.NoError(t, json.Unmarshal(raw, &file))
	build := func(ids []uint32, points [][3]float64, neighbours [][]uint32) []drawings.Tile {
		tiles := make([]drawings.Tile, len(ids))
		for i, id := range ids {
			links := []drawings.TileID{drawings.TileID(id + rimOffset)}
			for _, neighbour := range neighbours[i] {
				links = append(links, drawings.TileID(neighbour))
			}
			tiles[i] = drawings.Tile{
				ID: drawings.TileID(id), Neighbours: links,
				Point: drawings.Point{X: points[i][0], Y: points[i][1], Z: points[i][2]},
			}
		}
		return tiles
	}
	return build(file.Swastikas[0].IDs, file.Swastikas[0].Points, file.Swastikas[0].Neighbours),
		build(file.Others[0].IDs, file.Others[0].Points, file.Others[0].Neighbours)
}

// Planet: its ledger's takes, and its map, where every tile asked is answered, nobody's unless painted.
type stubPlanet struct {
	takes []paint.Take
	since time.Time
	tiles map[drawings.TileID]drawings.MapTile
	asked [][]drawings.TileID
	err   error
}

func (p *stubPlanet) Takes(_ context.Context, since, _ time.Time) ([]paint.Take, error) {
	p.since = since
	if p.err != nil {
		return nil, p.err
	}
	return p.takes, nil
}

func (p *stubPlanet) Tiles(_ context.Context, ids []drawings.TileID) ([]drawings.MapTile, error) {
	p.asked = append(p.asked, ids)
	answer := make([]drawings.MapTile, len(ids))
	for i, id := range ids {
		answer[i] = p.tiles[id]
		answer[i].ID = id
	}
	return answer, nil
}

func (p *stubPlanet) paint(tiles []drawings.Tile, flag string, holder cpsession.AccountID, rim string) {
	for _, tile := range tiles {
		p.tiles[tile.ID] = drawings.MapTile{Tile: tile, Flag: flag, Holder: holder}
		rimTile := drawings.TileID(uint32(tile.ID) + rimOffset)
		p.tiles[rimTile] = drawings.MapTile{Tile: drawings.Tile{ID: rimTile}, Flag: rim, Holder: cz}
	}
}

type reporter struct{ told []graffiti.Snapshot }

func (r *reporter) Report(_ context.Context, flagged graffiti.Graffiti) {
	r.told = append(r.told, flagged.TakeSnapshot())
}

type squad struct {
	clock  *clock
	planet *stubPlanet
	kept   *inmemory_graffiti_storage.Storage
	told   *reporter
	patrol *patrol_usecase.UseCase
}

func newSquad() squad {
	s := squad{
		clock:  &clock{now: now},
		planet: &stubPlanet{tiles: map[drawings.TileID]drawings.MapTile{}},
		kept:   inmemory_graffiti_storage.New(),
		told:   &reporter{},
	}
	s.patrol = patrol_usecase.New(s.planet, s.kept, s.told, s.clock,
		graffiti.NewPatrol(time.Hour, "https://clickplanet.lol/play", &graffiti.SequentialIDs{}))
	return s
}

func (s squad) took(account cpsession.AccountID, flag string, tiles []drawings.Tile) {
	for _, tile := range tiles {
		s.planet.takes = append(s.planet.takes,
			paint.Take{Account: account, Tile: tile.ID, Flag: flag, At: now.Add(-10 * time.Minute)})
	}
}

func TestAPatrolReadsTheLedgerOfTheLastHour(t *testing.T) {
	s := newSquad()

	require.NoError(t, s.patrol.Execute(t.Context()))

	assert.Equal(t, now.Add(-time.Hour), s.planet.since)
}

func TestASwastikaPaintedOnAnotherFlagsGroundIsFlaggedOnceAndToldOnce(t *testing.T) {
	swastika, _ := samples(t)
	s := newSquad()
	s.took(foe, "ro", swastika)
	s.planet.paint(swastika, "ro", foe, "cz")

	require.NoError(t, s.patrol.Execute(t.Context()))
	s.clock.now = now.Add(time.Minute)
	require.NoError(t, s.patrol.Execute(t.Context()))

	require.Len(t, s.told.told, 1, "seen again, it is not told again")
	flagged := s.told.told[0]
	assert.Equal(t, foe, flagged.Account)
	assert.Equal(t, "ro", flagged.Flag)
	assert.Equal(t, graffiti.Swastika, flagged.Kind)
	assert.Len(t, flagged.Tiles, len(swastika))
	assert.Equal(t, now, flagged.FlaggedAt)
	assert.Contains(t, flagged.Link, "https://clickplanet.lol/play?lat=")

	kept := s.kept.Kept()
	require.Len(t, kept, 1, "seen again, it is one graffiti")
	assert.Equal(t, now, kept[0].FlaggedAt)
	assert.Equal(t, now.Add(time.Minute), kept[0].SeenAt)
}

func TestPlayersPaintingOverASwastikaOnTheirOwnGroundAreNotFlagged(t *testing.T) {
	swastika, _ := samples(t)
	s := newSquad()
	s.took(cz, "cz", swastika)
	s.planet.paint(swastika, "cz", cz, "cz")

	require.NoError(t, s.patrol.Execute(t.Context()))

	assert.Empty(t, s.told.told)
	assert.Empty(t, s.kept.Kept())
}

func TestASwastikaNoLongerHeldByItsPainterIsNotFlagged(t *testing.T) {
	swastika, _ := samples(t)
	s := newSquad()
	s.took(foe, "ro", swastika)
	s.planet.paint(swastika, "ro", cz, "cz")

	require.NoError(t, s.patrol.Execute(t.Context()))

	assert.Empty(t, s.told.told, "another's takes hold the tiles: a revert, or the same flag painted again")
}

func TestAnOrdinaryDrawingIsNotFlaggedAndTheMapIsAskedOnce(t *testing.T) {
	_, other := samples(t)
	s := newSquad()
	s.took(foe, "ro", other)
	s.planet.paint(other, "ro", foe, "cz")

	require.NoError(t, s.patrol.Execute(t.Context()))

	assert.Empty(t, s.told.told)
	assert.Len(t, s.planet.asked, 1, "no rim is asked when nothing reads as a swastika")
}

func TestAPatrolOfNothingAsksNoTile(t *testing.T) {
	s := newSquad()

	require.NoError(t, s.patrol.Execute(t.Context()))

	assert.Empty(t, s.planet.asked)
}

func TestALedgerThatDoesNotAnswerFailsThePatrol(t *testing.T) {
	s := newSquad()
	s.planet.err = errors.New("planet is down")

	assert.ErrorContains(t, s.patrol.Execute(t.Context()), "planet is down")
	assert.Empty(t, s.told.told)
}

func TestAGraffitiNotKeptIsNotToldAndTheNextPatrolTellsIt(t *testing.T) {
	swastika, _ := samples(t)
	s := newSquad()
	s.took(foe, "ro", swastika)
	s.planet.paint(swastika, "ro", foe, "cz")
	s.kept.FailWith(errors.New("postgres is down"))

	assert.ErrorContains(t, s.patrol.Execute(t.Context()), "postgres is down")
	assert.Empty(t, s.told.told)

	s.kept.FailWith(nil)
	require.NoError(t, s.patrol.Execute(t.Context()))
	assert.Len(t, s.told.told, 1)
}
