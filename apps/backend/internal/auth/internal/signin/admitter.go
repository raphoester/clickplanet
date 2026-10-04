package signin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
)

type IdentityStore interface {
	accounts.SessionFinder
	accounts.AccountFinder
	Identity(ctx context.Context, provider string, subject string) (*accounts.Identity, error)
	AccountOfEmail(ctx context.Context, address string) (*accounts.Account, error)
	SaveSignIn(ctx context.Context, signIn accounts.SignIn) error
}

type Publisher interface {
	Publish(event proto.Message)
}

type Admitter struct {
	store    IdentityStore
	ids      accounts.IDProvider
	tokens   accounts.TokenGenerator
	lifetime accounts.Lifetime
	events   Publisher
}

func NewAdmitter(
	store IdentityStore, ids accounts.IDProvider, tokens accounts.TokenGenerator, lifetime accounts.Lifetime, events Publisher,
) *Admitter {
	return &Admitter{store: store, ids: ids, tokens: tokens, lifetime: lifetime.WithDefaults(), events: events}
}

type Visitor struct {
	Account  *accounts.Account
	Replaces accounts.TokenHash
}

func (a *Admitter) Visitor(ctx context.Context, cookieHeader string, now time.Time) (*Visitor, error) {
	session, err := accounts.Caller(ctx, a.store, cookieHeader, now)
	if errors.Is(err, accounts.ErrNoAccount) {
		return &Visitor{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find the caller: %w", err)
	}

	account, err := a.store.Account(ctx, session.Account)
	if errors.Is(err, accounts.ErrAccountNotFound) {
		return &Visitor{Replaces: session.TokenHash}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find the caller's account: %w", err)
	}
	return &Visitor{Account: account, Replaces: session.TokenHash}, nil
}

type Admission struct {
	Account   accounts.AccountID
	Outcome   accounts.Outcome
	SetCookie string
}

func (a *Admitter) Admit(
	ctx context.Context, provider string, intent accounts.Intent, claim accounts.Claim, visitor *Visitor, now time.Time,
) (*Admission, error) {
	admission, err := a.admit(ctx, provider, intent, claim, visitor, now)
	// Another browser linked this identity first; retrying finds it known.
	if errors.Is(err, accounts.ErrIdentityTaken) {
		admission, err = a.admit(ctx, provider, intent, claim, visitor, now)
	}
	return admission, err
}

func (a *Admitter) admit(
	ctx context.Context, provider string, intent accounts.Intent, claim accounts.Claim, visitor *Visitor, now time.Time,
) (*Admission, error) {
	known, err := a.store.Identity(ctx, provider, claim.Subject)
	if errors.Is(err, accounts.ErrIdentityNotFound) {
		known = nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to find the identity: %w", err)
	}

	var owner *accounts.Account
	if address := claim.VerifiedEmail(); known == nil && address != "" {
		owner, err = a.store.AccountOfEmail(ctx, address)
		if errors.Is(err, accounts.ErrAccountNotFound) {
			owner = nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to find the account of the address: %w", err)
		}
	}

	current := visitor.Account
	outcome, err := accounts.OutcomeOf(intent, current, known, owner, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to link %s: %w", provider, err)
	}
	signIn := accounts.SignIn{Replaces: visitor.Replaces}
	var account accounts.AccountID
	switch outcome {
	case accounts.SignedIn:
		account = known.Account
	case accounts.Linked:
		account = current.ID
	case accounts.Joined:
		account = owner.ID
	case accounts.Created:
		if account, err = a.ids.NewID(); err != nil {
			return nil, fmt.Errorf("failed to get an account id: %w", err)
		}
		signIn.NewAccount = true
	}
	if outcome != accounts.SignedIn {
		signIn.Identity = accounts.NewIdentity(provider, claim, account, now)
	}

	token, err := a.tokens.NewToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get a session token: %w", err)
	}
	signIn.Session = accounts.LinkedSession(account, token, a.lifetime, now)

	if err := a.store.SaveSignIn(ctx, signIn); err != nil {
		return nil, fmt.Errorf("failed to save the sign-in: %w", err)
	}

	signedIn := &authv1.SignedIn{AccountId: account.String()}
	if current != nil {
		signedIn.PreviousAccountId = current.ID.String()
	}
	// After the sign-in is saved: a subscriber moves what it keeps for the old account.
	a.events.Publish(signedIn)

	return &Admission{Account: account, Outcome: outcome, SetCookie: signIn.Session.Cookie(token, now)}, nil
}
