// Package get_profile_usecase reads the caller's profile.
package get_profile_usecase

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Profiles interface {
	Profile(account players.AccountID) (players.Profile, bool)
}

type UseCase struct {
	profiles Profiles
}

func New(profiles Profiles) *UseCase {
	return &UseCase{profiles: profiles}
}

// Execute answers a profile with no name for an account that never chose one.
func (u *UseCase) Execute(account players.AccountID) players.Profile {
	profile, ok := u.profiles.Profile(account)
	if !ok {
		return players.Profile{Account: account}
	}
	return profile
}
