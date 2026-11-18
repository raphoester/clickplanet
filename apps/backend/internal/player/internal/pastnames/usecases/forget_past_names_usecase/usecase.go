package forget_past_names_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Storage interface {
	Forget(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	storage Storage
}

func New(storage Storage) *UseCase {
	return &UseCase{storage: storage}
}

// Deletes the names an account's player left, on player.v1.PlayerForgotten: after the forget, not in it, since no read
// shows the names of an account with no profile.
func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	if err := u.storage.Forget(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the names the account's player left: %w", err)
	}
	return nil
}
