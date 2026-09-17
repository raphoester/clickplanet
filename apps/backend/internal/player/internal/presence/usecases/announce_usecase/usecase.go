// Package announce_usecase records that a player is playing, under which name, tag and flag.
package announce_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Profiles interface {
	Profile(ctx context.Context, account players.AccountID) (players.Profile, error)
}

type Visits interface {
	Record(visit presence.Visit)
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type In struct {
	Account   players.AccountID
	Country   string
	GuestName string
	IP        string
}

type UseCase struct {
	profiles  Profiles
	visits    Visits
	countries CountryChecker
	clock     cptime.Clock
	tagSalt   string
}

func New(profiles Profiles, visits Visits, countries CountryChecker, clock cptime.Clock, tagSalt string) *UseCase {
	return &UseCase{profiles: profiles, visits: visits, countries: countries, clock: clock, tagSalt: tagSalt}
}

// Execute reads the username on every announce, so a name chosen or changed shows within one interval.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	if !u.countries.CheckCountry(in.Country) {
		return fmt.Errorf("%w: %q", presence.ErrUnknownCountry, in.Country)
	}

	profile, err := u.profiles.Profile(ctx, in.Account)
	if err != nil && !errors.Is(err, players.ErrNoProfile) {
		return fmt.Errorf("failed to read the profile: %w", err)
	}

	u.visits.Record(presence.Visit{
		Account:   in.Account,
		Username:  profile.Name,
		Admin:     profile.Admin,
		GuestName: presence.GuestNameOf(in.GuestName),
		Tag:       players.TagOf(u.tagSalt, in.IP),
		Country:   in.Country,
		At:        u.clock.Now(),
	})

	return nil
}
