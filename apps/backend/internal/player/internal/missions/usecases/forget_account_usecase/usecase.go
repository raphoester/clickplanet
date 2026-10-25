package forget_account_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Sheets interface {
	DeleteAccount(ctx context.Context, account players.AccountID) error
}

func New(sheets Sheets) *UseCase {
	return &UseCase{sheets: sheets}
}

type UseCase struct {
	sheets Sheets
}

func (u *UseCase) Execute(ctx context.Context, account players.AccountID) error {
	if err := u.sheets.DeleteAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the account's missions: %w", err)
	}
	return nil
}
