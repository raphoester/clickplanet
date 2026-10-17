package start_admin_sign_in_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Out struct {
	AuthorizationURL string
	SetCookie        string
}

func New(providers signin.Providers, secrets signin.Secrets, sealer signin.Sealer, clock cptime.Clock) *UseCase {
	return &UseCase{providers: providers, secrets: secrets, sealer: sealer, clock: clock}
}

type UseCase struct {
	providers signin.Providers
	secrets   signin.Secrets
	sealer    signin.Sealer
	clock     cptime.Clock
}

func (u *UseCase) Execute(_ context.Context) (*Out, error) {
	if u.providers.Off() {
		return nil, adminsignin.ErrSignInOff
	}
	provider, err := u.providers.Provider(signin.Google)
	if err != nil {
		return nil, fmt.Errorf("failed to find Google: %w", err)
	}

	now := u.clock.Now()
	flow, err := signin.NewFlow(signin.Google, accounts.IntentSignIn, accounts.AccountID{}, u.secrets, now)
	if err != nil {
		return nil, fmt.Errorf("failed to start the flow: %w", err)
	}
	sealed, err := u.sealer.Sealed(flow)
	if err != nil {
		return nil, fmt.Errorf("failed to seal the flow: %w", err)
	}

	return &Out{AuthorizationURL: provider.AuthorizationURL(flow), SetCookie: adminsignin.FlowCookie(flow, sealed, now)}, nil
}
