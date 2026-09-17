// Package set_name_usecase cleans the name the caller chose, and keeps it.
package set_name_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Profiles interface {
	SaveProfile(ctx context.Context, profile players.Profile) error
}

type UseCase struct {
	profiles Profiles
	clock    cptime.Clock
}

func New(profiles Profiles, clock cptime.Clock) *UseCase {
	return &UseCase{profiles: profiles, clock: clock}
}

type In struct {
	Account players.AccountID
	Name    string
}

// Execute answers players.ErrInvalidName for a name that is empty or too long once cleaned.
func (u *UseCase) Execute(ctx context.Context, in In) (players.Profile, error) {
	name, err := players.NameOf(in.Name)
	if err != nil {
		return players.Profile{}, err //nolint:wrapcheck // the handler maps the domain's sentinel.
	}

	profile := players.Profile{Account: in.Account, Name: name, UpdatedAt: u.clock.Now()}
	if err := u.profiles.SaveProfile(ctx, profile); err != nil {
		return players.Profile{}, fmt.Errorf("failed to save the name: %w", err)
	}

	return profile, nil
}
