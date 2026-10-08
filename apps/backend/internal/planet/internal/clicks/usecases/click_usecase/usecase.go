package click_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type TilesChecker interface {
	CheckTile(tile uint32) bool
}

type Tiles interface {
	Click(ctx context.Context, tile uint32, flag string) (clicks.Impact, error)
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

	// Never on the wire: a dropped click answers the zero Out, so it would expose a ban.
	Outcome clicks.Outcome
}

type IUseCase interface {
	Execute(ctx context.Context, in In) (Out, error)
}

func New(tilesChecker TilesChecker, tiles Tiles, countryChecker CountryChecker) *UseCase {
	return &UseCase{tilesChecker: tilesChecker, tiles: tiles, countryChecker: countryChecker}
}

type UseCase struct {
	tilesChecker   TilesChecker
	tiles          Tiles
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

	impact, err := u.tiles.Click(ctx, in.TileID, in.CountryID)
	if err != nil {
		return Out{}, fmt.Errorf("failed to set tile: %w", err)
	}

	return Out{Outcome: impact.Outcome}, nil
}
