package rpc_planet_map

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/drawings"
)

type Planet interface {
	GetTiles(ctx context.Context, req *connect.Request[planetv1.GetTilesRequest]) (*connect.Response[planetv1.GetTilesResponse], error)
}

const (
	askTimeout = 10 * time.Second
	// Planet answers 20,000 at most.
	batch = 10_000
)

func New(planet Planet) Map {
	return Map{planet: planet}
}

type Map struct {
	planet Planet
}

func (m Map) Tiles(ctx context.Context, ids []drawings.TileID) ([]drawings.MapTile, error) {
	tiles := make([]drawings.MapTile, 0, len(ids))
	for start := 0; start < len(ids); start += batch {
		asked := make([]uint32, 0, batch)
		for _, id := range ids[start:min(start+batch, len(ids))] {
			asked = append(asked, uint32(id))
		}
		answer, err := m.ask(ctx, asked)
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

func (m Map) ask(ctx context.Context, asked []uint32) ([]*planetv1.MapTile, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := m.planet.GetTiles(ctx, connect.NewRequest(&planetv1.GetTilesRequest{TileIds: asked}))
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
