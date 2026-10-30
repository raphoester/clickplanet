package complete_crazygames_sign_in_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type In struct {
	Token        string
	CookieHeader string
	Landing      arrivals.Landing
}

type Out = signin.Admission

type UseCase struct {
	providers signin.TokenProviders
	admitter  *signin.Admitter
	clock     cptime.Clock
}

func New(providers signin.TokenProviders, admitter *signin.Admitter, clock cptime.Clock) *UseCase {
	return &UseCase{providers: providers, admitter: admitter, clock: clock}
}

// Always a sign-in, never a link: CrazyGames offers no other way in, so its account is the one the player plays on.
func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	provider, err := u.providers.Provider(signin.CrazyGames)
	if err != nil {
		return nil, fmt.Errorf("failed to find the provider: %w", err)
	}
	claim, err := provider.Claim(ctx, in.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to read the token: %w", err)
	}

	now := u.clock.Now()
	visitor, err := u.admitter.Visitor(ctx, in.CookieHeader, in.Landing, now)
	if err != nil {
		return nil, fmt.Errorf("failed to find the browser's account: %w", err)
	}

	admission, err := u.admitter.Admit(ctx, signin.CrazyGames, accounts.IntentSignIn, *claim, visitor, now)
	if err != nil {
		return nil, fmt.Errorf("failed to sign in: %w", err)
	}
	return admission, nil
}
