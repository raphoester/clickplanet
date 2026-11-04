package forget_nameplate_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type Nameplates interface {
	Forget(ctx context.Context, account accounts.AccountID) error
}

type UseCase struct {
	nameplates Nameplates
}

func New(nameplates Nameplates) *UseCase {
	return &UseCase{nameplates: nameplates}
}

func (u *UseCase) Execute(ctx context.Context, account accounts.AccountID) error {
	if err := u.nameplates.Forget(ctx, account); err != nil {
		return fmt.Errorf("failed to forget the nameplate: %w", err)
	}
	return nil
}
