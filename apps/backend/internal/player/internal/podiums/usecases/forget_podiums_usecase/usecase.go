package forget_podiums_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Places interface {
	DeleteAccount(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	places Places
}

func New(places Places) *UseCase {
	return &UseCase{places: places}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	if err := u.places.DeleteAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account's podium places: %w", err)
	}
	return nil
}
