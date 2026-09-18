// Package announce_usecase records that a player is playing, under which name and flag.
package announce_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

// Authors says who an account is, and gives a guest its code the first time: get_author_usecase.
type Authors interface {
	Execute(ctx context.Context, account players.AccountID) (players.Author, error)
}

type Visits interface {
	Record(visit presence.Visit)
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type In struct {
	Account players.AccountID
	Country string
	IP      string
}

type UseCase struct {
	authors   Authors
	visits    Visits
	countries CountryChecker
	clock     cptime.Clock
	tagSalt   string
}

func New(authors Authors, visits Visits, countries CountryChecker, clock cptime.Clock, tagSalt string) *UseCase {
	return &UseCase{authors: authors, visits: visits, countries: countries, clock: clock, tagSalt: tagSalt}
}

// Execute reads the name on every announce, so a username chosen or changed shows within one interval. The
// address is kept on the visit as a tag, for the storage's cap, and never shown.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	if !u.countries.CheckCountry(in.Country) {
		return fmt.Errorf("%w: %q", presence.ErrUnknownCountry, in.Country)
	}

	author, err := u.authors.Execute(ctx, in.Account)
	if err != nil {
		return fmt.Errorf("failed to name the player: %w", err)
	}

	u.visits.Record(presence.Visit{
		Account: in.Account,
		Author:  author,
		Tag:     players.TagOf(u.tagSalt, in.IP),
		Country: in.Country,
		At:      u.clock.Now(),
	})

	return nil
}
