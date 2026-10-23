package drop_bomb_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var ErrNoBomb = errors.New("no bomb to drop")

type Bombs interface {
	SpendBomb(ctx context.Context, holder bonuses.Holder) error
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

func (u *UseCase) Execute(ctx context.Context, in In) (clicks.Blast, error) {
	if !u.countries.CheckCountry(in.CountryID) {
		return clicks.Blast{}, fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, in.CountryID)
	}

	tile, arc := u.geography.Nearest(in.Target)
	if tile == 0 {
		return clicks.Blast{}, fmt.Errorf("%w: a target with no direction", clicks.ErrTileOutOfRange)
	}

	if err := u.bombs.SpendBomb(ctx, bonuses.HolderOf(clicks.PayerOf(ctx))); err != nil {
		if errors.Is(err, bonuses.ErrNotHeld) {
			return clicks.Blast{}, ErrNoBomb
		}
		return clicks.Blast{}, err
	}

	blast, err := u.clearer.Clear(ctx, u.rules.Blast(in.CountryID, in.Target, tile, arc, u.geography))
	if err != nil {
		return clicks.Blast{}, fmt.Errorf("failed to clear the blast at tile %d: %w", tile, err)
	}

	return blast, nil
}
