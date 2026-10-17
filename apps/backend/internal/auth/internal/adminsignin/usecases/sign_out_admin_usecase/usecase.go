package sign_out_admin_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
)

type Deleter interface {
	DeleteSession(ctx context.Context, tokenHash accounts.TokenHash) error
}

func New(deleter Deleter) *UseCase {
	return &UseCase{deleter: deleter}
}

type UseCase struct {
	deleter Deleter
}

func (u *UseCase) Execute(ctx context.Context, cookieHeader string) error {
	value, found := accounts.CookieValue(cookieHeader, adminsignin.CookieName)
	if !found {
		return nil
	}

	if err := u.deleter.DeleteSession(ctx, accounts.TokenOf(value).Hash()); err != nil {
		return fmt.Errorf("failed to end the admin session: %w", err)
	}
	return nil
}
