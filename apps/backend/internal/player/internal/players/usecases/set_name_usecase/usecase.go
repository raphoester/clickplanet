// Package set_name_usecase cleans the name the caller chose, and keeps it.
package set_name_usecase

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Profiles interface {
	SaveProfile(profile players.Profile)
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
func (u *UseCase) Execute(in In) (players.Profile, error) {
	name, err := players.NameOf(in.Name)
	if err != nil {
		return players.Profile{}, err //nolint:wrapcheck // the handler maps the domain's sentinel.
	}

	profile := players.Profile{Account: in.Account, Name: name, UpdatedAt: u.clock.Now()}
	u.profiles.SaveProfile(profile)

	return profile, nil
}
