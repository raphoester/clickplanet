// Package spread_click is the spread bonus: each click it has left also takes
// every tile touching the one clicked.
//
// The server picks those tiles off its own map. A client that named them would
// be a client that could name any tiles it liked, which is the whole of the cheat.
package spread_click

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

// Spreads spends one click of a caller's spread charge, and says whether there was one.
type Spreads interface {
	SpendSpreadClick(holder bonuses.Holder) bool
}

// Neighbours is the part of clicks.Geography this reads.
type Neighbours interface {
	Neighbours(id uint32) []uint32
}

type TileStorage interface {
	Set(ctx context.Context, tile uint32, value string) error
}

// Publisher tells the planet a click spread, so every client can show it.
type Publisher interface {
	PublishSpread(spread bonuses.Spread)
}

func New(implementation click_usecase.IUseCase, spreads Spreads, neighbours Neighbours, storage TileStorage, publisher Publisher) *UseCase {
	return &UseCase{
		implementation: implementation,
		spreads:        spreads,
		neighbours:     neighbours,
		storage:        storage,
		publisher:      publisher,
	}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	spreads        Spreads
	neighbours     Neighbours
	storage        TileStorage
	publisher      Publisher
}

// Execute spreads only a click the rule accepted, so a refused country or tile
// spreads nothing and costs no spread click. The neighbours need no check of their own: the map only
// holds real tiles, and the country is the one the rule just accepted.
//
// A tile with no neighbours — one of the lone islands — takes itself and
// nothing else. That is what the map says, and the bonus does not pretend
// otherwise.
func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	out, err := u.implementation.Execute(ctx, in)
	if err != nil || !u.spreads.SpendSpreadClick(bonuses.HolderOf(clicks.PayerOf(ctx))) {
		return out, err
	}

	neighbours := u.neighbours.Neighbours(in.TileID)
	for _, neighbour := range neighbours {
		if err := u.storage.Set(ctx, neighbour, in.CountryID); err != nil {
			return out, fmt.Errorf("failed to spread onto tile %d: %w", neighbour, err)
		}
	}

	// After the tiles, so no client shows a spread the map has not taken. The
	// slice is the map's own table, so it is copied before it leaves.
	u.publisher.PublishSpread(bonuses.Spread{
		CountryID:  in.CountryID,
		Tile:       in.TileID,
		Neighbours: append([]uint32(nil), neighbours...),
	})

	return out, nil
}
