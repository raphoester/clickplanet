// Package move_visit_usecase moves a browser's visit to the account it signed in to, so the roster shows the
// player at once, under its username, and never beside the guest it was.
package move_visit_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Profiles interface {
	Profile(ctx context.Context, account players.AccountID) (players.Profile, error)
}

type Visits interface {
	Move(from, to players.AccountID, username players.Name)
}

type UseCase struct {
	profiles Profiles
	visits   Visits
}

func New(profiles Profiles, visits Visits) *UseCase {
	return &UseCase{profiles: profiles, visits: visits}
}

// Execute moves the visit of from to to. A sign-in that keeps the browser on its account changes nothing:
// a guest that links its first identity has no username yet, and a signed-in account keeps the one it had.
// Reading the profile then could only race SetName, and put back the name it replaced.
func (u *UseCase) Execute(ctx context.Context, from, to players.AccountID) error {
	if from == to {
		return nil
	}

	profile, err := u.profiles.Profile(ctx, to)
	if err != nil && !errors.Is(err, players.ErrNoProfile) {
		return fmt.Errorf("failed to read the profile: %w", err)
	}

	u.visits.Move(from, to, profile.Name)
	return nil
}
