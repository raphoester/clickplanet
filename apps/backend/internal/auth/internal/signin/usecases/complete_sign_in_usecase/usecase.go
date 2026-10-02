// Package complete_sign_in_usecase finishes a sign-in: it checks the flow, asks the provider who signed in, and signs the browser in to that identity's account.
package complete_sign_in_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store = signin.IdentityStore

type Publisher = signin.Publisher

type In struct {
	Code         string
	State        string
	CookieHeader string
}

type Out = signin.Admission

type UseCase struct {
	providers signin.Providers
	sealer    signin.Sealer
	admitter  *signin.Admitter
	clock     cptime.Clock
}

func New(
	providers signin.Providers,
	sealer signin.Sealer,
	store Store,
	ids accounts.IDProvider,
	tokens accounts.TokenGenerator,
	lifetime accounts.Lifetime,
	events Publisher,
	clock cptime.Clock,
) *UseCase {
	return &UseCase{
		providers: providers,
		sealer:    sealer,
		admitter:  signin.NewAdmitter(store, ids, tokens, lifetime, events),
		clock:     clock,
	}
}

// Execute answers signin.ErrSignInOff, signin.ErrFlowInvalid or signin.ErrProviderRefused for a sign-in it cannot finish,
// and accounts.ErrIdentityLinkedElsewhere or accounts.ErrProviderAlreadyLinked for a link it refuses, having written nothing.
func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	if u.providers.Off() {
		return nil, signin.ErrSignInOff
	}

	now := u.clock.Now()
	flow, provider, err := u.flow(in, now)
	if err != nil {
		return nil, err
	}

	visitor, err := u.admitter.Visitor(ctx, in.CookieHeader, now)
	if err != nil {
		return nil, fmt.Errorf("failed to find the browser's account: %w", err)
	}
	if err := flow.AccountError(visitor.Account); err != nil {
		return nil, fmt.Errorf("failed to check the account: %w", err)
	}

	claim, err := provider.Exchange(ctx, in.Code, flow)
	if err != nil {
		return nil, fmt.Errorf("failed to ask %s who signed in: %w", flow.Provider, err)
	}

	admission, err := u.admitter.Admit(ctx, flow.Provider, flow.Intent, *claim, visitor, now)
	if err != nil {
		return nil, fmt.Errorf("failed to sign in: %w", err)
	}
	return admission, nil
}

func (u *UseCase) flow(in In, now time.Time) (*signin.Flow, signin.Provider, error) {
	sealed, found := accounts.CookieValue(in.CookieHeader, signin.FlowCookieName)
	if !found {
		return nil, nil, fmt.Errorf("%w: the browser sent no %s cookie", signin.ErrFlowInvalid, signin.FlowCookieName)
	}
	flow, err := u.sealer.Opened(sealed)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open the flow: %w", err)
	}
	if err := flow.CallbackError(in.State, now); err != nil {
		return nil, nil, fmt.Errorf("failed to check the flow: %w", err)
	}

	provider, err := u.providers.Provider(flow.Provider)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", signin.ErrFlowInvalid, err)
	}
	return flow, provider, nil
}
