package place_defender_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var (
	ErrNoDefender = errors.New("no defender to place")
	ErrNotYours   = errors.New("the tile does not wear the flag")
	ErrFull       = errors.New("the tile holds as many defenders as it can")
)

type Defenders interface {
	SpendDefender(holder bonuses.Holder) bool
	Held(holder bonuses.Holder) bonuses.Held
}

type Owners interface {
	Owner(tile uint32) (string, bool)
}

type Garrisons interface {
	Full(tile uint32, country string) bool
	Reinforce(tile uint32, country string)
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

type In struct {
	TileID    uint32
	CountryID string

	Dud bool
}

func New(defenders Defenders, owners Owners, garrisons Garrisons, countries CountryChecker) *UseCase {
	return &UseCase{defenders: defenders, owners: owners, garrisons: garrisons, countries: countries}
}

type UseCase struct {
	defenders Defenders
	owners    Owners
	garrisons Garrisons
	countries CountryChecker
}

func (u *UseCase) Execute(ctx context.Context, in In) (bonuses.Held, error) {
	if !u.countries.CheckCountry(in.CountryID) {
		return bonuses.Held{}, fmt.Errorf("%w: %q", clicks.ErrUnknownCountry, in.CountryID)
	}

	owner, ok := u.owners.Owner(in.TileID)
	if !ok {
		return bonuses.Held{}, fmt.Errorf("%w: %d", clicks.ErrTileOutOfRange, in.TileID)
	}

	if owner != in.CountryID {
		return bonuses.Held{}, ErrNotYours
	}

	if u.garrisons.Full(in.TileID, in.CountryID) {
		return bonuses.Held{}, ErrFull
	}

	holder := bonuses.HolderOf(clicks.PayerOf(ctx))
	if !u.defenders.SpendDefender(holder) {
		return bonuses.Held{}, ErrNoDefender
	}

	if !in.Dud {
		u.garrisons.Reinforce(in.TileID, in.CountryID)
	}

	return u.defenders.Held(holder), nil
}
