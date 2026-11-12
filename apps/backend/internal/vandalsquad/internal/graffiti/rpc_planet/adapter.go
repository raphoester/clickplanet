package rpc_planet

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/paint"
)

type Client interface {
	GetTiles(ctx context.Context, req *connect.Request[planetv1.GetTilesRequest]) (*connect.Response[planetv1.GetTilesResponse], error)
	GetLedger(ctx context.Context, req *connect.Request[planetv1.GetLedgerRequest]) (*connect.Response[planetv1.GetLedgerResponse], error)
}

const (
	askTimeout = 10 * time.Second
	// Planet answers 20,000 at most.
	batch = 10_000
)

func New(client Client) Planet {
	return Planet{client: client}
}

type Planet struct {
	client Client
}

// Every tile a player took between since and until, oldest first, page by page as the ledger answers.
func (p Planet) Takes(ctx context.Context, since, until time.Time) ([]paint.Take, error) {
	var takes []paint.Take
	page := ""
	for {
		answer, err := p.ledger(ctx, since, until, page)
		if err != nil {
			return nil, err
		}
		for _, change := range answer.GetChanges() {
			take, ok, err := takeOf(change)
			if err != nil {
				return nil, err
			}
			if ok {
				takes = append(takes, take)
			}
		}
		if answer.GetNextPage() == "" {
			return takes, nil
		}
		page = answer.GetNextPage()
	}
}

func (p Planet) ledger(ctx context.Context, since, until time.Time, page string) (*planetv1.GetLedgerResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := p.client.GetLedger(ctx, connect.NewRequest(&planetv1.GetLedgerRequest{
		Since: timestamppb.New(since), Until: timestamppb.New(until), Page: page,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to read planet's ledger since %s: %w", since.Format(time.RFC3339), err)
	}
	return res.Msg, nil
}

// A strike, a shield, a bomb's clear and a change no account made took no tile for a player.
func takeOf(change *planetv1.LedgerChange) (paint.Take, bool, error) {
	update := change.GetUpdate()
	if change.GetAccountId() == "" || update.GetCountryId() == "" || update.GetCountryId() == update.GetPreviousCountryId() {
		return paint.Take{}, false, nil
	}
	account, err := uuid.Parse(change.GetAccountId())
	if err != nil {
		return paint.Take{}, false, fmt.Errorf("planet's ledger named a taker %q: %w", change.GetAccountId(), err)
	}
	return paint.Take{
		Account: cpsession.AccountID(account),
		Tile:    drawings.TileID(update.GetTileId()),
		Flag:    update.GetCountryId(),
		At:      change.GetAt().AsTime(),
	}, true, nil
}

func (p Planet) Tiles(ctx context.Context, ids []drawings.TileID) ([]drawings.MapTile, error) {
	tiles := make([]drawings.MapTile, 0, len(ids))
	for start := 0; start < len(ids); start += batch {
		asked := make([]uint32, 0, batch)
		for _, id := range ids[start:min(start+batch, len(ids))] {
			asked = append(asked, uint32(id))
		}
		answer, err := p.ask(ctx, asked)
		if err != nil {
			return nil, err
		}
		for _, tile := range answer {
			read, err := tileOf(tile)
			if err != nil {
				return nil, err
			}
			tiles = append(tiles, read)
		}
	}
	return tiles, nil
}

func (p Planet) ask(ctx context.Context, asked []uint32) ([]*planetv1.MapTile, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := p.client.GetTiles(ctx, connect.NewRequest(&planetv1.GetTilesRequest{TileIds: asked}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask planet %d tiles: %w", len(asked), err)
	}
	if got := len(res.Msg.GetTiles()); got != len(asked) {
		return nil, fmt.Errorf("planet answered %d tiles of the %d asked", got, len(asked))
	}
	return res.Msg.GetTiles(), nil
}

func tileOf(tile *planetv1.MapTile) (drawings.MapTile, error) {
	holder := cpsession.NoAccount
	if id := tile.GetHolderAccountId(); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			return drawings.MapTile{}, fmt.Errorf("planet named tile %d's holder %q: %w", tile.GetTileId(), id, err)
		}
		holder = cpsession.AccountID(parsed)
	}
	neighbours := make([]drawings.TileID, len(tile.GetNeighbours()))
	for i, neighbour := range tile.GetNeighbours() {
		neighbours[i] = drawings.TileID(neighbour)
	}
	point := tile.GetPoint()
	return drawings.MapTile{
		Tile: drawings.Tile{
			ID:         drawings.TileID(tile.GetTileId()),
			Point:      drawings.Point{X: point.GetX(), Y: point.GetY(), Z: point.GetZ()},
			Neighbours: neighbours,
		},
		Flag:   tile.GetCountryId(),
		Holder: holder,
	}, nil
}
