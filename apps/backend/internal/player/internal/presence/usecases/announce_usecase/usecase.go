package announce_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Introducer interface {
	Execute(ctx context.Context, account players.AccountID) (players.Player, error)
}

type Visits interface {
	Record(visit presence.Visit)
}

// The flags the game ships with and the ones opened since: a closed one is known.
type CountryChecker interface {
	Known(ctx context.Context, flag string) (bool, error)
}

type In struct {
	Account players.AccountID
	Country string
	IP      string
}

type UseCase struct {
	introducer Introducer
	visits     Visits
	countries  CountryChecker
	clock      cptime.Clock
	tagSalt    string
}

func New(introducer Introducer, visits Visits, countries CountryChecker, clock cptime.Clock, tagSalt string) *UseCase {
	return &UseCase{introducer: introducer, visits: visits, countries: countries, clock: clock, tagSalt: tagSalt}
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	known, err := u.countries.Known(ctx, in.Country)
	if err != nil {
		return fmt.Errorf("failed to read whether %q is a flag: %w", in.Country, err)
	}
	if !known {
		return fmt.Errorf("%w: %q", presence.ErrUnknownCountry, in.Country)
	}

	player, err := u.introducer.Execute(ctx, in.Account)
	if err != nil {
		return fmt.Errorf("failed to name the player: %w", err)
	}

	u.visits.Record(presence.NewVisit(in.Account, player, players.TagOf(u.tagSalt, in.IP), in.Country, u.clock.Now()))

	return nil
}
