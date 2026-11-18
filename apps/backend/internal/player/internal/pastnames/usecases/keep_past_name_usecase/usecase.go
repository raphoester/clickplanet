package keep_past_name_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/pastnames"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Profiles interface {
	Profile(ctx context.Context, account players.AccountID) (players.Profile, error)
}

type UseCase struct {
	profiles Profiles
	storage  pastnames.Storage
}

func New(profiles Profiles, storage pastnames.Storage) *UseCase {
	return &UseCase{profiles: profiles, storage: storage}
}

// Keeps the name a player left, on player.v1.NameLeft. An account that holds no profile any more was forgotten since
// it left the name, and its names went with it: nothing is kept.
func (u *UseCase) Execute(ctx context.Context, name pastnames.PastName) error {
	_, err := u.profiles.Profile(ctx, name.TakeSnapshot().Account)
	if errors.Is(err, players.ErrNoProfile) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read whether the account still has a profile: %w", err)
	}
	if err := u.storage.Record(ctx, name); err != nil {
		return fmt.Errorf("failed to keep the name left: %w", err)
	}
	return nil
}
