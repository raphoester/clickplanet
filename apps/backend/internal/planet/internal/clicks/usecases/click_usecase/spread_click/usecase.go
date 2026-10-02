package spread_click

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Spreads interface {
	SpendSpreadClick(holder bonuses.Holder) bool
}

type Neighbours interface {
	Neighbours(id uint32) []uint32
}

type TileStorage interface {
	Owner(tile uint32) (string, bool)
	Set(ctx context.Context, tile uint32, value string) error
}

type Rule interface {
	Outcome(tile uint32, owner, flag string) clicks.Outcome
}

type Publisher interface {
	PublishSpread(spread bonuses.Spread)
}

func New(
	implementation click_usecase.IUseCase,
	spreads Spreads,
	neighbours Neighbours,
	storage TileStorage,
	rule Rule,
	publisher Publisher,
) *UseCase {
	return &UseCase{
		implementation: implementation,
		spreads:        spreads,
		neighbours:     neighbours,
		storage:        storage,
		rule:           rule,
		publisher:      publisher,
	}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	spreads        Spreads
	neighbours     Neighbours
	storage        TileStorage
	rule           Rule
	publisher      Publisher
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	out, err := u.implementation.Execute(ctx, in)
	if err != nil || !in.Spread || !u.spreads.SpendSpreadClick(bonuses.HolderOf(clicks.PayerOf(ctx))) {
		return out, err
	}

	neighbours := u.neighbours.Neighbours(in.TileID)
	for _, neighbour := range neighbours {
		owner, _ := u.storage.Owner(neighbour)
		after := u.rule.Outcome(neighbour, owner, in.CountryID).OwnerAfter(owner, in.CountryID)

		if err := u.storage.Set(ctx, neighbour, after); err != nil {
			return out, fmt.Errorf("failed to spread onto tile %d: %w", neighbour, err)
		}
	}

	// The neighbours are copied: Neighbours hands back the map's own table.
	u.publisher.PublishSpread(bonuses.Spread{
		CountryID:  in.CountryID,
		Tile:       in.TileID,
		Neighbours: append([]uint32(nil), neighbours...),
	})

	return out, nil
}
