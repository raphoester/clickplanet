package start_sign_in_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type In struct {
	Provider     string
	Intent       accounts.Intent
	CookieHeader string
}

type Out struct {
	AuthorizationURL string
	SetCookie        string
}

type UseCase struct {
	providers signin.Providers
	sessions  accounts.SessionFinder
	secrets   signin.Secrets
	sealer    signin.Sealer
	clock     cptime.Clock
}

func New(providers signin.Providers, sessions accounts.SessionFinder, secrets signin.Secrets, sealer signin.Sealer, clock cptime.Clock) *UseCase {
	return &UseCase{providers: providers, sessions: sessions, secrets: secrets, sealer: sealer, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	provider, err := u.providers.Provider(in.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to find the provider: %w", err)
	}

	now := u.clock.Now()
	account, err := signin.LinkTarget(ctx, u.sessions, in.Intent, in.CookieHeader, now)
	if err != nil {
		return nil, fmt.Errorf("failed to start the sign-in: %w", err)
	}

	flow, err := signin.NewFlow(in.Provider, in.Intent, account, u.secrets, now)
	if err != nil {
		return nil, fmt.Errorf("failed to start the flow: %w", err)
	}
	sealed, err := u.sealer.Sealed(flow)
	if err != nil {
		return nil, fmt.Errorf("failed to seal the flow: %w", err)
	}

	return &Out{AuthorizationURL: provider.AuthorizationURL(flow), SetCookie: flow.Cookie(sealed, now)}, nil
}
