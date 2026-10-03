package forget_worn_title_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Choices interface {
	DeleteAccount(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	choices Choices
}

func New(choices Choices) *UseCase {
	return &UseCase{choices: choices}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	if err := u.choices.DeleteAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account's worn title: %w", err)
	}
	return nil
}
