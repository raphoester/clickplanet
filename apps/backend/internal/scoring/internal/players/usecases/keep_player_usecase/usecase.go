package keep_player_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/players"
)

type Players interface {
	Keep(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	players Players
}

func New(players Players) *UseCase {
	return &UseCase{players: players}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	if err := u.players.Keep(ctx, account); err != nil {
		return fmt.Errorf("failed to keep the player: %w", err)
	}
	return nil
}
