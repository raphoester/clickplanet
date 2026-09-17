// Package get_profile_usecase reads the caller's profile.
package get_profile_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Profiles interface {
	Profile(ctx context.Context, account players.AccountID) (players.Profile, error)
}

type UseCase struct {
	profiles Profiles
}

func New(profiles Profiles) *UseCase {
	return &UseCase{profiles: profiles}
}

// Execute answers a profile with no name for an account that never chose one.
func (u *UseCase) Execute(ctx context.Context, account players.AccountID) (players.Profile, error) {
	profile, err := u.profiles.Profile(ctx, account)
	if errors.Is(err, players.ErrNoProfile) {
		return players.Profile{Account: account}, nil
	}
	if err != nil {
		return players.Profile{}, fmt.Errorf("failed to read the profile: %w", err)
	}
	return profile, nil
}
