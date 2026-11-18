package rpc_planet_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti/rpc_planet"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/paint"
)

const ana = "01900000-0000-7000-8000-000000000001"

var at = time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)

type planet struct {
	asked  [][]uint32
	short  bool
	board  string
	pages  []*planetv1.GetLedgerResponse
	ledger []*planetv1.GetLedgerRequest
}

func (p *planet) GetTiles(_ context.Context, req *connect.Request[planetv1.GetTilesRequest]) (*connect.Response[planetv1.GetTilesResponse], error) {
	p.asked = append(p.asked, req.Msg.GetTileIds())
	var answer []*planetv1.MapTile
	for _, id := range req.Msg.GetTileIds() {
		answer = append(answer, &planetv1.MapTile{
			TileId: id, CountryId: "ro", HolderAccountId: ana,
			Point: &planetv1.GlobePoint{X: 1, Y: 2, Z: 3}, Neighbours: []uint32{id + 1},
		})
	}
	if p.short {
		answer = answer[1:]
	}
	return connect.NewResponse(&planetv1.GetTilesResponse{Tiles: answer, Board: p.board}), nil
}

func (p *planet) GetLedger(_ context.Context, req *connect.Request[planetv1.GetLedgerRequest]) (*connect.Response[planetv1.GetLedgerResponse], error) {
	p.ledger = append(p.ledger, req.Msg)
	answer := p.pages[0]
	p.pages = p.pages[1:]
	return connect.NewResponse(answer), nil
}

func change(account string, tile uint32, country, previous string, shields uint32) *planetv1.LedgerChange {
	return &planetv1.LedgerChange{
		At: timestamppb.New(at), AccountId: account,
		Update: &planetv1.TileUpdate{TileId: tile, CountryId: country, PreviousCountryId: previous, Shields: shields},
	}
}

func TestEachTileIsReadInTheMapsWords(t *testing.T) {
	tiles, err := rpc_planet.New(&planet{}).Tiles(t.Context(), "earth-1", []drawings.TileID{7})

	require.NoError(t, err)
	assert.Equal(t, []drawings.MapTile{{
		Tile:   drawings.Tile{ID: 7, Point: drawings.Point{X: 1, Y: 2, Z: 3}, Neighbours: []drawings.TileID{8}},
		Flag:   "ro",
		Holder: cpsession.AccountID(uuid.MustParse(ana)),
	}}, tiles)
}

func TestManyTilesAreAskedInBatchesPlanetTakes(t *testing.T) {
	ids := make([]drawings.TileID, 25_000)
	for i := range ids {
		ids[i] = drawings.TileID(i + 1) //nolint:gosec // a few thousand.
	}
	asked := &planet{}

	tiles, err := rpc_planet.New(asked).Tiles(t.Context(), "earth-1", ids)

	require.NoError(t, err)
	assert.Len(t, tiles, 25_000)
	require.Len(t, asked.asked, 3)
	assert.Len(t, asked.asked[0], 10_000)
	assert.Len(t, asked.asked[2], 5_000)
}

func TestAnAnswerShortOfATileIsAnError(t *testing.T) {
	_, err := rpc_planet.New(&planet{short: true}).Tiles(t.Context(), "earth-1", []drawings.TileID{7, 8})

	assert.ErrorContains(t, err, "planet answered 1 tiles of the 2 asked")
}

func TestTheTakesAreEveryTileAPlayerTookOnEveryPageOfTheLedger(t *testing.T) {
	asked := &planet{pages: []*planetv1.GetLedgerResponse{
		{Board: "earth-1", Changes: []*planetv1.LedgerChange{
			change(ana, 1, "ro", "cz", 0),
			change(ana, 2, "ro", "", 0),
		}, NextPage: "20000"},
		{Board: "earth-1", Changes: []*planetv1.LedgerChange{change(ana, 3, "fr", "ro", 0)}},
	}}

	board, takes, err := rpc_planet.New(asked).Takes(t.Context(), at.Add(-time.Hour), at)

	require.NoError(t, err)
	assert.Equal(t, drawings.Board("earth-1"), board)
	account := cpsession.AccountID(uuid.MustParse(ana))
	assert.Equal(t, []paint.Take{
		{Account: account, Tile: 1, Flag: "ro", At: at},
		{Account: account, Tile: 2, Flag: "ro", At: at},
		{Account: account, Tile: 3, Flag: "fr", At: at},
	}, takes)
	require.Len(t, asked.ledger, 2)
	assert.Equal(t, at.Add(-time.Hour), asked.ledger[0].GetSince().AsTime())
	assert.Equal(t, at, asked.ledger[0].GetUntil().AsTime())
	assert.Empty(t, asked.ledger[0].GetPage())
	assert.Equal(t, "20000", asked.ledger[1].GetPage())
}

func TestAStrikeAShieldAClearOrAChangeOfNoAccountIsNoTake(t *testing.T) {
	asked := &planet{pages: []*planetv1.GetLedgerResponse{{Board: "earth-1", Changes: []*planetv1.LedgerChange{
		change(ana, 1, "cz", "cz", 3),
		change(ana, 2, "ro", "ro", 5),
		change(ana, 3, "", "ro", 0),
		change("", 4, "ro", "cz", 0),
	}}}}

	_, takes, err := rpc_planet.New(asked).Takes(t.Context(), at.Add(-time.Hour), at)

	require.NoError(t, err)
	assert.Empty(t, takes)
}

func TestALedgerNamingANonsenseTakerIsAnError(t *testing.T) {
	asked := &planet{pages: []*planetv1.GetLedgerResponse{{Board: "earth-1", Changes: []*planetv1.LedgerChange{change("nobody", 1, "ro", "cz", 0)}}}}

	_, _, err := rpc_planet.New(asked).Takes(t.Context(), at.Add(-time.Hour), at)

	assert.ErrorContains(t, err, `planet's ledger named a taker "nobody"`)
}

func TestALedgerReadWhosePagesNameTwoBoardsIsRefused(t *testing.T) {
	asked := &planet{pages: []*planetv1.GetLedgerResponse{
		{Board: "earth-1", Changes: []*planetv1.LedgerChange{change(ana, 1, "ro", "cz", 0)}, NextPage: "20000"},
		{Board: "earth-2", Changes: []*planetv1.LedgerChange{change(ana, 3, "fr", "ro", 0)}},
	}}

	_, _, err := rpc_planet.New(asked).Takes(t.Context(), at.Add(-time.Hour), at)

	assert.ErrorIs(t, err, rpc_planet.ErrBoardMoved, "tile ids of two boards are no one drawing")
}

func TestTilesOfAnotherBoardThanTheOnesAskedAreRefused(t *testing.T) {
	_, err := rpc_planet.New(&planet{board: "fixture-1"}).Tiles(t.Context(), "earth-1", []drawings.TileID{7})

	require.ErrorIs(t, err, drawings.ErrOtherBoard, "an arena plays a board of its own")

	_, err = rpc_planet.New(&planet{board: "earth-1"}).Tiles(t.Context(), "earth-1", []drawings.TileID{7})
	require.NoError(t, err)
}
