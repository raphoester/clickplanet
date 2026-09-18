// Package drop_bomb_usecase spends a caller's bomb where they aimed: it clears the tiles around the one hit,
// or, in the sea, nothing.
package drop_bomb_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// ErrNoBomb covers never won, already dropped and held too long, for the reason ErrNoSuchBonus does.
var ErrNoBomb = errors.New("no bomb to drop")

// Bombs spends the bomb charge a caller holds, and says whether there was one.
type Bombs interface {
	SpendBomb(holder bonuses.Holder) bool
}

type Map interface {
	bonuses.Ground
	Nearest(point clicks.Vec3) (uint32, float64)
}

type Clearer interface {
	Clear(ctx context.Context, blast clicks.Blast) (clicks.Blast, error)
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type In struct {
	Target    clicks.Vec3
	CountryID string

	// Dud is set by antibot_drop_bomb for a banned caller: the bomb is spent, clears nothing and is shown to nobody.
	Dud bool
}

func New(bombs Bombs, geography Map, clearer Clearer, countries CountryChecker, rules bonuses.BombRules) *UseCase {
	return &UseCase{
		bombs:     bombs,
		geography: geography,
		clearer:   clearer,
		countries: countries,
		rules:     rules,
	}
}

type UseCase struct {
	bombs     Bombs
	geography Map
	clearer   Clearer
	countries CountryChecker
	rules     bonuses.BombRules
}

// Execute checks the request before it takes the bomb, so a malformed drop does not cost one.
func (u *UseCase) Execute(ctx context.Context, in In) (clicks.Blast, error) {
	if !u.countries.CheckCountry(in.CountryID) {
		return clicks.Blast{}, fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, in.CountryID)
	}

	tile, arc := u.geography.Nearest(in.Target)
	if tile == 0 {
		return clicks.Blast{}, fmt.Errorf("%w: a target with no direction", clicks.ErrTileOutOfRange)
	}

	if !u.bombs.SpendBomb(bonuses.HolderOf(clicks.PayerOf(ctx))) {
		return clicks.Blast{}, ErrNoBomb
	}

	if in.Dud {
		return clicks.Blast{}, nil
	}

	blast, err := u.clearer.Clear(ctx, u.rules.Blast(in.CountryID, in.Target, tile, arc, u.geography))
	if err != nil {
		return clicks.Blast{}, fmt.Errorf("failed to clear the blast at tile %d: %w", tile, err)
	}

	return blast, nil
}
