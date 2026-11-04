package move_visit_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Introducer interface {
	Execute(ctx context.Context, account players.AccountID) (players.Player, error)
}

type Visits interface {
	Move(from, to players.AccountID, player players.Player)
}

type UseCase struct {
	introducer Introducer
	visits     Visits
}

func New(introducer Introducer, visits Visits) *UseCase {
	return &UseCase{introducer: introducer, visits: visits}
}

func (u *UseCase) Execute(ctx context.Context, from, to players.AccountID) error {
	// Same account: re-reading its name here could race SetName and restore the old one.
	if from == to {
		return nil
	}

	player, err := u.introducer.Execute(ctx, to)
	if err != nil {
		return fmt.Errorf("failed to name the account: %w", err)
	}

	u.visits.Move(from, to, player)
	return nil
}
