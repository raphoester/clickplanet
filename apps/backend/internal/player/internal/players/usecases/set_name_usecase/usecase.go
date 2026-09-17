// Package set_name_usecase checks the username the caller chose, and keeps it.
package set_name_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Profiles interface {
	SaveProfile(ctx context.Context, profile players.Profile) error
}

// Accounts is the auth module, asked whether an account signed in with a provider.
type Accounts interface {
	Linked(ctx context.Context, account players.AccountID) (bool, error)
}

type UseCase struct {
	profiles Profiles
	accounts Accounts
	clock    cptime.Clock
}

func New(profiles Profiles, accounts Accounts, clock cptime.Clock) *UseCase {
	return &UseCase{profiles: profiles, accounts: accounts, clock: clock}
}

type In struct {
	Account players.AccountID
	Name    string
}

// Execute answers players.ErrInvalidName for a name that breaks a rule, players.ErrNotLinked for a guest,
// and players.ErrNameTaken for a name another account holds. The name is checked first, so a name no
// account may hold costs no call to auth.
func (u *UseCase) Execute(ctx context.Context, in In) (players.Profile, error) {
	name, err := players.NameOf(in.Name)
	if err != nil {
		return players.Profile{}, err //nolint:wrapcheck // the handler maps the domain's sentinel.
	}

	linked, err := u.accounts.Linked(ctx, in.Account)
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to ask whether the account is linked: %w", err)
	}
	if !linked {
		return players.Profile{}, players.ErrNotLinked
	}

	profile := players.Profile{Account: in.Account, Name: name, UpdatedAt: u.clock.Now()}
	err = u.profiles.SaveProfile(ctx, profile)
	if errors.Is(err, players.ErrNameTaken) {
		return players.Profile{}, err //nolint:wrapcheck // the handler maps the port's sentinel.
	}
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to save the name: %w", err)
	}

	return profile, nil
}
