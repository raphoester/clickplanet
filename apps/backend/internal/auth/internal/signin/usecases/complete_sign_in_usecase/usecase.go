// Package complete_sign_in_usecase finishes a sign-in: it checks the flow, asks the provider who signed in, and signs the browser in to that identity's account.
package complete_sign_in_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	accounts.SessionFinder
	accounts.AccountFinder
	FindIdentity(ctx context.Context, provider string, subject string) (*accounts.Identity, error)
	SaveSignIn(ctx context.Context, signIn accounts.SignIn) error
}

type In struct {
	Code         string
	State        string
	CookieHeader string
}

// Out is the account the browser is now on, and its new session cookie.
type Out struct {
	Account   uuid.UUID
	Outcome   accounts.Outcome
	SetCookie string
}

type UseCase struct {
	providers signin.Providers
	sealer    signin.Sealer
	store     Store
	ids       accounts.IDProvider
	tokens    accounts.TokenGenerator
	lifetime  accounts.Lifetime
	clock     cptime.Clock
}

func New(
	providers signin.Providers,
	sealer signin.Sealer,
	store Store,
	ids accounts.IDProvider,
	tokens accounts.TokenGenerator,
	lifetime accounts.Lifetime,
	clock cptime.Clock,
) *UseCase {
	return &UseCase{
		providers: providers,
		sealer:    sealer,
		store:     store,
		ids:       ids,
		tokens:    tokens,
		lifetime:  lifetime.WithDefaults(),
		clock:     clock,
	}
}

// Execute answers signin.ErrSignInOff, signin.ErrFlowInvalid or signin.ErrProviderRefused for a sign-in it cannot finish.
func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	if u.providers.Off() {
		return nil, signin.ErrSignInOff
	}

	now := u.clock.Now()
	flow, provider, err := u.flow(in, now)
	if err != nil {
		return nil, err
	}

	claim, err := provider.Exchange(ctx, in.Code, flow)
	if err != nil {
		return nil, fmt.Errorf("failed to ask %s who signed in: %w", flow.Provider, err)
	}

	current, replaces, err := u.current(ctx, in.CookieHeader, now)
	if err != nil {
		return nil, err
	}

	out, err := u.signIn(ctx, flow.Provider, claim, current, replaces, now)
	if errors.Is(err, accounts.ErrIdentityTaken) {
		// Another browser linked the same identity a moment ago: it is known now.
		out, err = u.signIn(ctx, flow.Provider, claim, current, replaces, now)
	}
	return out, err
}

func (u *UseCase) flow(in In, now time.Time) (*signin.Flow, signin.Provider, error) {
	sealed, found := accounts.CookieValue(in.CookieHeader, signin.FlowCookieName)
	if !found {
		return nil, nil, fmt.Errorf("%w: the browser sent no %s cookie", signin.ErrFlowInvalid, signin.FlowCookieName)
	}
	flow, err := u.sealer.Open(sealed)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open the flow: %w", err)
	}
	if err := flow.Check(in.State, now); err != nil {
		return nil, nil, fmt.Errorf("failed to check the flow: %w", err)
	}

	provider, err := u.providers.Get(flow.Provider)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", signin.ErrFlowInvalid, err)
	}
	return flow, provider, nil
}

// current is the account the browser is on and its session to replace, or nothing when it has no live session.
func (u *UseCase) current(ctx context.Context, cookieHeader string, now time.Time) (*accounts.Account, []byte, error) {
	session, err := accounts.Caller(ctx, u.store, cookieHeader, now)
	if errors.Is(err, accounts.ErrNoAccount) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find the caller: %w", err)
	}

	account, err := u.store.FindAccount(ctx, session.Account)
	if errors.Is(err, accounts.ErrAccountNotFound) {
		return nil, session.TokenHash, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find the caller's account: %w", err)
	}
	return account, session.TokenHash, nil
}

func (u *UseCase) signIn(
	ctx context.Context, provider string, claim *accounts.Claim, current *accounts.Account, replaces []byte, now time.Time,
) (*Out, error) {
	known, err := u.store.FindIdentity(ctx, provider, claim.Subject)
	if errors.Is(err, accounts.ErrIdentityNotFound) {
		known = nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to find the identity: %w", err)
	}

	outcome := accounts.Choose(current, known, provider)
	signIn := accounts.SignIn{Replaces: replaces}
	var account uuid.UUID
	switch outcome {
	case accounts.SignedIn:
		account = known.Account
	case accounts.Linked:
		account = current.ID
	case accounts.Created:
		if account, err = u.ids.NewID(); err != nil {
			return nil, fmt.Errorf("failed to get an account id: %w", err)
		}
		signIn.NewAccount = true
	}
	if outcome != accounts.SignedIn {
		signIn.Identity = accounts.NewIdentity(provider, *claim, account, now)
	}

	token, err := u.tokens.NewToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get a session token: %w", err)
	}
	signIn.Session = accounts.StartLinked(account, token, u.lifetime, now)

	if err := u.store.SaveSignIn(ctx, signIn); err != nil {
		return nil, fmt.Errorf("failed to save the sign-in: %w", err)
	}
	return &Out{Account: account, Outcome: outcome, SetCookie: signIn.Session.Cookie(token, now)}, nil
}
