// Package sign_out_usecase ends the session a browser's cookie holds, and nothing else.
package sign_out_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type SessionDeleter interface {
	DeleteSession(ctx context.Context, tokenHash []byte) error
}

type UseCase struct {
	sessions SessionDeleter
}

func New(sessions SessionDeleter) *UseCase {
	return &UseCase{sessions: sessions}
}

// Execute answers the Set-Cookie that clears the session. A browser with no session is signed out already.
func (u *UseCase) Execute(ctx context.Context, cookieHeader string) (string, error) {
	token, err := accounts.TokenFromCookies(cookieHeader)
	if errors.Is(err, accounts.ErrNoSessionCookie) {
		return accounts.ClearCookie(), nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read the cookie: %w", err)
	}

	if err := u.sessions.DeleteSession(ctx, token.Hash); err != nil {
		return "", fmt.Errorf("failed to delete the session: %w", err)
	}
	return accounts.ClearCookie(), nil
}
