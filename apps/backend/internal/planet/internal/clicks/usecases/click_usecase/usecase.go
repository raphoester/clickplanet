package click_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type TilesChecker interface {
	CheckTile(tile uint32) bool
}

type TileStorage interface {
	Click(ctx context.Context, tile uint32, value string) error
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type In struct {
	TileID    uint32
	CountryID string

	Spread  bool
	Enclose bool
}

type Out struct {
	Budget clicks.Budget

	Limited bool

	Gift bool
}

type IUseCase interface {
	Execute(ctx context.Context, in In) (Out, error)
}

func New(
	tilesChecker TilesChecker,
	tileStorage TileStorage,
	countryChecker CountryChecker,
) *UseCase {
	return &UseCase{
		tilesChecker:   tilesChecker,
		tileStorage:    tileStorage,
		countryChecker: countryChecker,
	}
}

type UseCase struct {
	tilesChecker   TilesChecker
	tileStorage    TileStorage
	countryChecker CountryChecker
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	if in.Spread && in.Enclose {
		return Out{}, clicks.ErrBonusesTogether
	}

	if !u.countryChecker.CheckCountry(in.CountryID) {
		return Out{}, fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, in.CountryID)
	}

	if !u.tilesChecker.CheckTile(in.TileID) {
		return Out{}, fmt.Errorf("%w: %d", clicks.ErrTileOutOfRange, in.TileID)
	}

	if err := u.tileStorage.Click(ctx, in.TileID, in.CountryID); err != nil {
		return Out{}, fmt.Errorf("failed to set tile: %w", err)
	}

	return Out{}, nil
}
