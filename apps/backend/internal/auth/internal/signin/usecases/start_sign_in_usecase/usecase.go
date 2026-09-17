// Package start_sign_in_usecase opens a sign-in: a new flow, sealed in the browser's cookie, and the provider URL to send it to.
package start_sign_in_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Out struct {
	AuthorizationURL string
	SetCookie        string
}

type UseCase struct {
	providers signin.Providers
	secrets   signin.Secrets
	sealer    signin.Sealer
	clock     cptime.Clock
}

func New(providers signin.Providers, secrets signin.Secrets, sealer signin.Sealer, clock cptime.Clock) *UseCase {
	return &UseCase{providers: providers, secrets: secrets, sealer: sealer, clock: clock}
}

// Execute answers signin.ErrSignInOff or signin.ErrUnknownProvider for a provider it does not offer.
func (u *UseCase) Execute(_ context.Context, providerName string) (*Out, error) {
	provider, err := u.providers.Provider(providerName)
	if err != nil {
		return nil, fmt.Errorf("failed to find the provider: %w", err)
	}

	now := u.clock.Now()
	flow, err := signin.NewFlow(providerName, u.secrets, now)
	if err != nil {
		return nil, fmt.Errorf("failed to start the flow: %w", err)
	}
	sealed, err := u.sealer.Sealed(flow)
	if err != nil {
		return nil, fmt.Errorf("failed to seal the flow: %w", err)
	}

	return &Out{AuthorizationURL: provider.AuthorizationURL(flow), SetCookie: flow.Cookie(sealed, now)}, nil
}
