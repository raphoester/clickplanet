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
	SaveSignIn(ctx context.Context, signIn accounts.SignIn) error
}

type Publisher interface {
	Publish(event proto.Message)
}

// Admitter signs a browser in to an identity it proved it holds, with a provider or with an emailed code.
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

// Visitor is the account a browser is on and the session a sign-in replaces. Both are empty with no live session.
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

// Admission is the account the browser is now on, and its new session cookie.
type Admission struct {
	Account   accounts.AccountID
	Outcome   accounts.Outcome
	SetCookie string
}

// Admit answers accounts.ErrIdentityLinkedElsewhere or accounts.ErrProviderAlreadyLinked for a link it refuses, having written nothing.
func (a *Admitter) Admit(
	ctx context.Context, provider string, intent accounts.Intent, claim accounts.Claim, visitor *Visitor, now time.Time,
) (*Admission, error) {
	admission, err := a.admit(ctx, provider, intent, claim, visitor, now)
	if errors.Is(err, accounts.ErrIdentityTaken) {
		// Another browser linked the same identity a moment ago: it is known now.
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

	current := visitor.Account
	outcome, err := accounts.OutcomeOf(intent, current, known, provider)
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

	// After the sign-in is saved: a subscriber moves what it keeps for the browser's old account.
	signedIn := &authv1.SignedIn{AccountId: account.String()}
	if current != nil {
		signedIn.PreviousAccountId = current.ID.String()
	}
	a.events.Publish(signedIn)

	return &Admission{Account: account, Outcome: outcome, SetCookie: signIn.Session.Cookie(token, now)}, nil
}
