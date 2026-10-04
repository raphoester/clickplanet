package get_caller_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type UseCase struct {
	sessions accounts.SessionFinder
	clock    cptime.Clock
}

func New(sessions accounts.SessionFinder, clock cptime.Clock) *UseCase {
	return &UseCase{sessions: sessions, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, cookieHeader string) (accounts.AccountID, error) {
	session, err := accounts.Caller(ctx, u.sessions, cookieHeader, u.clock.Now())
	if err != nil {
		return accounts.AccountID{}, fmt.Errorf("failed to find the caller: %w", err)
	}
	return session.Account, nil
}
