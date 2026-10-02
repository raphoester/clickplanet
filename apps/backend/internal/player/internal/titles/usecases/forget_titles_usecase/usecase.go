package forget_titles_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Titles interface {
	DeleteAccount(ctx context.Context, account players.AccountID) error
}

type UseCase struct {
	titles Titles
}

func New(titles Titles) *UseCase {
	return &UseCase{titles: titles}
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	if err := u.titles.DeleteAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account's titles: %w", err)
	}
	return nil
}
