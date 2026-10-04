package name_account_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Names interface {
	Assign(ctx context.Context, account players.AccountID, at time.Time) error
}

type Profiles interface {
	Profile(ctx context.Context, account players.AccountID) (players.Profile, error)
}

type UseCase struct {
	names    Names
	profiles Profiles
	clock    cptime.Clock
}

func New(names Names, profiles Profiles, clock cptime.Clock) *UseCase {
	return &UseCase{names: names, profiles: profiles, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) (players.Profile, error) {
	if err := u.names.Assign(ctx, account, u.clock.Now()); err != nil {
		return players.Profile{}, fmt.Errorf("failed to name the account: %w", err)
	}
	profile, err := u.profiles.Profile(ctx, account)
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to read the account's name: %w", err)
	}
	return profile, nil
}
