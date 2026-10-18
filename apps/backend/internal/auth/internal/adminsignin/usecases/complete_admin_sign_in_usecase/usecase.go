package complete_admin_sign_in_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/adminsignin"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type In struct {
	Code         string
	State        string
	CookieHeader string
}

type Out struct {
	Account   accounts.AccountID
	SetCookie string
}

func New(
	providers signin.Providers, sealer signin.Sealer, store adminsignin.Store, issuer *adminsignin.Issuer, clock cptime.Clock,
) *UseCase {
	return &UseCase{providers: providers, sealer: sealer, store: store, issuer: issuer, clock: clock}
}

type UseCase struct {
	providers signin.Providers
	sealer    signin.Sealer
	store     adminsignin.Store
	issuer    *adminsignin.Issuer
	clock     cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	if u.providers.Off() {
		return nil, adminsignin.ErrSignInOff
	}

	now := u.clock.Now()
	flow, provider, err := u.flow(in, now)
	if err != nil {
		return nil, err
	}

	claim, err := provider.Exchange(ctx, in.Code, flow)
	if err != nil {
		return nil, fmt.Errorf("failed to ask Google who signed in: %w", err)
	}

	candidate, err := u.store.Candidate(ctx, signin.Google, claim.Subject())
	if errors.Is(err, adminsignin.ErrUnknownIdentity) {
		return nil, fmt.Errorf("%w: %w", adminsignin.ErrNotAnOperator, err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find who signed in: %w", err)
	}

	session, err := u.issuer.Session(candidate, now)
	if err != nil {
		return nil, fmt.Errorf("failed to open an admin session: %w", err)
	}
	kept := session.TakeSnapshot()
	if err := u.store.SaveSession(ctx, kept); err != nil {
		return nil, fmt.Errorf("failed to keep the admin session: %w", err)
	}

	return &Out{Account: kept.Account, SetCookie: session.Cookie(now)}, nil
}

func (u *UseCase) flow(in In, now time.Time) (*signin.Flow, signin.Provider, error) {
	sealed, found := accounts.CookieValue(in.CookieHeader, adminsignin.FlowCookieName)
	if !found {
		return nil, nil, fmt.Errorf("%w: the browser sent no %s cookie", signin.ErrFlowInvalid, adminsignin.FlowCookieName)
	}
	flow, err := u.sealer.Opened(sealed)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open the flow: %w", err)
	}
	if err := flow.CallbackError(in.State, now); err != nil {
		return nil, nil, fmt.Errorf("failed to check the flow: %w", err)
	}
	if flow.Provider() != signin.Google {
		return nil, nil, fmt.Errorf("%w: only Google opens an admin session, not %q", signin.ErrFlowInvalid, flow.Provider())
	}

	provider, err := u.providers.Provider(signin.Google)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", signin.ErrFlowInvalid, err)
	}
	return flow, provider, nil
}
