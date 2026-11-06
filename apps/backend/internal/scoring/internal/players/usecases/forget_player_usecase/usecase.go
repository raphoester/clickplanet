package forget_player_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/players"
)

type Players interface {
	Forget(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	players Players
}

func New(players Players) *UseCase {
	return &UseCase{players: players}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	if err := u.players.Forget(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the player: %w", err)
	}
	return nil
}
