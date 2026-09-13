// Package click is the one use case that writes. It validates the country and
// the tile before handing the tile over, and it is the only thing in the clicks
// context that may change the map.
package click

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/toll"
)

// The three ports below are this use case's own, declared here and nowhere
// else: a use case names what it needs, so a dependency added to one never
// widens the others. Only the click path validates a country, and only the
// click path writes, which is why neither interface is shared with the reads.
type TilesChecker interface {
	CheckTile(tile uint32) bool
}

type TileStorage interface {
	Set(ctx context.Context, tile uint32, value string) error
	// SetBoosted is Set, with the update it publishes marked as boosted.
	SetBoosted(ctx context.Context, tile uint32, value string) error
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type In struct {
	TileID    uint32
	CountryID string

	// Boosted is set by throttle_click when the caller's allowance is boosted,
	// so the update the click publishes says so and every client can show it.
	Boosted bool
}

// Out is what a click answers with beyond having happened. It carries no game
// state — the map is read back over the stream — only the caller's remaining
// allowance, which throttle_click fills in and the rule below leaves unset.
//
// It is a return value and not something left on the context because the
// throttle is now part of this chain: a decorator can answer its caller, and a
// value that has to be smuggled past one is a sign the policy is in the wrong
// place.
type Out struct {
	Budget toll.Budget

	// Limited says whether anything throttles clicks at all, which is not the
	// same answer as an allowance of zero.
	Limited bool
}

// IUseCase is the click chain: the rule below, and whatever decorates it.
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
	if !u.countryChecker.CheckCountry(in.CountryID) {
		return Out{}, fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, in.CountryID)
	}

	if !u.tilesChecker.CheckTile(in.TileID) {
		return Out{}, fmt.Errorf("%w: %d", clicks.ErrTileOutOfRange, in.TileID)
	}

	set := u.tileStorage.Set
	if in.Boosted {
		set = u.tileStorage.SetBoosted
	}

	if err := set(ctx, in.TileID, in.CountryID); err != nil {
		return Out{}, fmt.Errorf("failed to set tile: %w", err)
	}

	return Out{}, nil
}
