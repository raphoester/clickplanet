package anonymize_arrivals_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
)

func New(kept arrivals.Store) *UseCase {
	return &UseCase{arrivals: kept}
}

type UseCase struct {
	arrivals arrivals.Store
}

func (u *UseCase) Execute(ctx context.Context, account arrivals.AccountID) error {
	found, err := u.arrivals.Of(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the account's arrivals: %w", err)
	}

	if err := u.arrivals.Save(ctx, arrivals.AllAnonymized(found)...); err != nil {
		return fmt.Errorf("failed to anonymize the account's arrivals: %w", err)
	}
	return nil
}
