package drop_bomb_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var ErrNoBomb = errors.New("no bomb to drop")

type Bombs interface {
	Spend(ctx context.Context, holder charges.Holder, after func(charges.Held) (charges.Held, bool)) error
}

type Map interface {
	charges.Ground
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

func New(bombs Bombs, geography Map, clearer Clearer, countries CountryChecker, rules charges.Bombing) *UseCase {
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
	rules     charges.Bombing
}

// Execute refuses a drop while bombs are off before anything else, so the bomb stays held: DropBomb has no throttle,
// and no token is spent.
func (u *UseCase) Execute(ctx context.Context, in In) (clicks.Blast, error) {
	if err := u.rules.Refuses(); err != nil {
		return clicks.Blast{}, err //nolint:wrapcheck // the sentinel, for the handler.
	}
	if !u.countries.CheckCountry(in.CountryID) {
		return clicks.Blast{}, fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, in.CountryID)
	}

	tile, arc := u.geography.Nearest(in.Target)
	if tile == 0 {
		return clicks.Blast{}, fmt.Errorf("%w: a target with no direction", clicks.ErrTileOutOfRange)
	}

	if err := u.bombs.Spend(ctx, charges.HolderOf(clicks.PayerOf(ctx)), charges.Held.AfterBomb); err != nil {
		if errors.Is(err, charges.ErrNotHeld) {
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
