package set_shown_in_clips_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Profiles interface {
	SaveShownInClips(ctx context.Context, account players.AccountID, shown bool) error
}

type UseCase struct {
	profiles Profiles
}

func New(profiles Profiles) *UseCase {
	return &UseCase{profiles: profiles}
}

type In struct {
	Account players.AccountID
	Shown   bool
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	err := u.profiles.SaveShownInClips(ctx, in.Account, in.Shown)
	if errors.Is(err, players.ErrNoProfile) {
		return err //nolint:wrapcheck // the handler maps the port's sentinel.
	}
	if err != nil {
		return fmt.Errorf("failed to save whether the clips show the name: %w", err)
	}
	return nil
}
