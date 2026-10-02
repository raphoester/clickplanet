package complete_sign_in_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	accounts.SessionFinder
	accounts.AccountFinder
	Identity(ctx context.Context, provider string, subject string) (*accounts.Identity, error)
	SaveSignIn(ctx context.Context, signIn accounts.SignIn) error
}

type Publisher interface {
	Publish(event proto.Message)
}

type In struct {
	Code         string
	State        string
	CookieHeader string
}

type Out struct {
	Account   accounts.AccountID
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
	events    Publisher
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
		store:     store,
		ids:       ids,
		tokens:    tokens,
		lifetime:  lifetime.WithDefaults(),
		events:    events,
		clock:     clock,
	}
}

func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	if u.providers.Off() {
		return nil, signin.ErrSignInOff
	}

	now := u.clock.Now()
	flow, provider, err := u.flow(in, now)
	if err != nil {
		return nil, err
	}

	current, replaces, err := u.current(ctx, in.CookieHeader, now)
	if err != nil {
		return nil, err
	}
	if err := flow.AccountError(current); err != nil {
		return nil, fmt.Errorf("failed to check the account: %w", err)
	}

	claim, err := provider.Exchange(ctx, in.Code, flow)
	if err != nil {
		return nil, fmt.Errorf("failed to ask %s who signed in: %w", flow.Provider, err)
	}

	out, err := u.signIn(ctx, flow, claim, current, replaces, now)
	// Another browser linked this identity first; retrying finds it known.
	if errors.Is(err, accounts.ErrIdentityTaken) {
		out, err = u.signIn(ctx, flow, claim, current, replaces, now)
	}
	return out, err
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

func (u *UseCase) current(ctx context.Context, cookieHeader string, now time.Time) (*accounts.Account, accounts.TokenHash, error) {
	session, err := accounts.Caller(ctx, u.store, cookieHeader, now)
	if errors.Is(err, accounts.ErrNoAccount) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find the caller: %w", err)
	}

	account, err := u.store.Account(ctx, session.Account)
	if errors.Is(err, accounts.ErrAccountNotFound) {
		return nil, session.TokenHash, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find the caller's account: %w", err)
	}
	return account, session.TokenHash, nil
}

func (u *UseCase) signIn(
	ctx context.Context, flow *signin.Flow, claim *accounts.Claim, current *accounts.Account, replaces accounts.TokenHash, now time.Time,
) (*Out, error) {
	provider := flow.Provider
	known, err := u.store.Identity(ctx, provider, claim.Subject)
	if errors.Is(err, accounts.ErrIdentityNotFound) {
		known = nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to find the identity: %w", err)
	}

	outcome, err := accounts.OutcomeOf(flow.Intent, current, known, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to link %s: %w", provider, err)
	}
	signIn := accounts.SignIn{Replaces: replaces}
	var account accounts.AccountID
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
	signIn.Session = accounts.LinkedSession(account, token, u.lifetime, now)

	if err := u.store.SaveSignIn(ctx, signIn); err != nil {
		return nil, fmt.Errorf("failed to save the sign-in: %w", err)
	}

	signedIn := &authv1.SignedIn{AccountId: account.String()}
	if current != nil {
		signedIn.PreviousAccountId = current.ID.String()
	}
	u.events.Publish(signedIn)

	return &Out{Account: account, Outcome: outcome, SetCookie: signIn.Session.Cookie(token, now)}, nil
}
