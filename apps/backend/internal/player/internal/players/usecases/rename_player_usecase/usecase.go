package rename_player_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Profiles interface {
	Profile(ctx context.Context, account players.AccountID) (players.Profile, error)
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

func (u *UseCase) Execute(ctx context.Context, in In) (players.Profile, error) {
	name, err := players.NameOf(in.Name)
	if err != nil {
		return players.Profile{}, err //nolint:wrapcheck // the handler maps the domain's sentinel.
	}

	current, err := u.profiles.Profile(ctx, in.Account)
	if errors.Is(err, players.ErrNoProfile) {
		return players.Profile{}, err //nolint:wrapcheck // the handler maps the port's sentinel.
	}
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to read the profile: %w", err)
	}

	profile := current.RenamedByOperator(name, u.clock.Now())
	err = u.profiles.SaveProfile(ctx, profile)
	if errors.Is(err, players.ErrNameTaken) {
		return players.Profile{}, err //nolint:wrapcheck // the handler maps the port's sentinel.
	}
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to save the name: %w", err)
	}

	return profile, nil
}
