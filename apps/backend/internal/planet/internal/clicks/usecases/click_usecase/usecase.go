// Package click_usecase is the one use case that writes. It validates the country and
// the tile, then writes what the home-soil rule says the click leaves on it: the
// flag, or nobody on another country's native ground. It is the only thing in the
// clicks context that may change the map.
package click_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// The ports below are this use case's own, declared here and nowhere
// else: a use case names what it needs, so a dependency added to one never
// widens the others. Only the click path validates a country, and only the
// click path writes, which is why neither interface is shared with the reads.
type TilesChecker interface {
	CheckTile(tile uint32) bool
}

type TileStorage interface {
	Owner(tile uint32) (string, bool)
	Set(ctx context.Context, tile uint32, value string) error
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

// Rule is the home-soil rule: what a click for flag does to a tile owner holds. clicks.HomeSoil is one.
type Rule interface {
	Outcome(tile uint32, owner, flag string) clicks.Outcome
}

type In struct {
	TileID    uint32
	CountryID string

	// What the player switched on for this click. A charge is used only when the player chooses:
	// spread_click spends a spread click only with Spread, and enclose_click an enclosure only with Enclose.
	// One at most: both is refused with clicks.ErrBonusesTogether, before anything is written or spent.
	Spread  bool
	Enclose bool
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
	Budget clicks.Budget

	// Limited says whether anything throttles clicks at all, which is not the
	// same answer as an allowance of zero.
	Limited bool

	// Outcome is what the rule did to the tile clicked, for the decorators
	// inside the shadow ban: enclose_click closes a shape only with a tile
	// taken, and prom_click counts the clears. It never reaches the wire: a
	// dropped click answers the zero Out, so an outcome on the answer would
	// tell a banned caller its clicks are dropped.
	Outcome clicks.Outcome
}

// IUseCase is the click chain: the rule below, and whatever decorates it.
type IUseCase interface {
	Execute(ctx context.Context, in In) (Out, error)
}

func New(
	tilesChecker TilesChecker,
	tileStorage TileStorage,
	countryChecker CountryChecker,
	rule Rule,
) *UseCase {
	return &UseCase{
		tilesChecker:   tilesChecker,
		tileStorage:    tileStorage,
		countryChecker: countryChecker,
		rule:           rule,
	}
}

type UseCase struct {
	tilesChecker   TilesChecker
	tileStorage    TileStorage
	countryChecker CountryChecker
	rule           Rule
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

	// Read apart from the write, like the ledger's Previous: a click racing this one on the same tile can
	// leave it cleared where it would now be taken. Both are one click's worth, and the next click settles it.
	owner, _ := u.tileStorage.Owner(in.TileID)
	outcome := u.rule.Outcome(in.TileID, owner, in.CountryID)

	// A clear is a write like a take: it costs the click, publishes a TileUpdate with no country, and lands in
	// the ledger. Unchanged writes the owner back, which Set leaves alone.
	if err := u.tileStorage.Set(ctx, in.TileID, outcome.OwnerAfter(owner, in.CountryID)); err != nil {
		return Out{}, fmt.Errorf("failed to set tile: %w", err)
	}

	return Out{Outcome: outcome}, nil
}
