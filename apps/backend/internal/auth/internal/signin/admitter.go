package signin

import (
	"context"
	"errors"
	"fmt"
	"time"

	authv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/auth/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/arrivals"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type IdentityStore interface {
	accounts.SessionFinder
	accounts.AccountFinder
	Identity(ctx context.Context, provider string, subject string) (*accounts.Identity, error)
	AccountOfEmail(ctx context.Context, address string) (*accounts.Account, error)
	SaveSignIn(ctx context.Context, signIn accounts.SignIn) error
}

type ArrivalRecorder interface {
	Record(ctx context.Context, account accounts.AccountID, landing arrivals.Landing, now time.Time) error
}

type Admitter struct {
	transactioner cpatomic.Transactioner
	store         IdentityStore
	outbox        cpatomic.Outbox
	arrivals      ArrivalRecorder
	ids           accounts.IDProvider
	tokens        accounts.TokenGenerator
	lifetime      accounts.Lifetime
}

func NewAdmitter(
	transactioner cpatomic.Transactioner,
	store IdentityStore,
	outbox cpatomic.Outbox,
	arrivals ArrivalRecorder,
	ids accounts.IDProvider,
	tokens accounts.TokenGenerator,
	lifetime accounts.Lifetime,
) *Admitter {
	return &Admitter{
		transactioner: transactioner, store: store, outbox: outbox, arrivals: arrivals, ids: ids, tokens: tokens,
		lifetime: lifetime.WithDefaults(),
	}
}

type Visitor struct {
	account  *accounts.Account
	replaces accounts.TokenHash
	landing  arrivals.Landing
}

func (v *Visitor) Account() *accounts.Account {
	return v.account
}

func (a *Admitter) Visitor(ctx context.Context, cookieHeader string, landing arrivals.Landing, now time.Time) (*Visitor, error) {
	session, err := accounts.Caller(ctx, a.store, cookieHeader, now)
	if errors.Is(err, accounts.ErrNoAccount) {
		return &Visitor{landing: landing}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find the caller: %w", err)
	}

	account, err := a.store.Account(ctx, session.Account())
	if errors.Is(err, accounts.ErrAccountNotFound) {
		return &Visitor{replaces: session.TokenHash(), landing: landing}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find the caller's account: %w", err)
	}
	return &Visitor{account: account, replaces: session.TokenHash(), landing: landing}, nil
}

type Admission struct {
	account   accounts.AccountID
	outcome   accounts.Outcome
	setCookie string
}

func (a *Admission) Account() accounts.AccountID {
	return a.account
}

func (a *Admission) Outcome() accounts.Outcome {
	return a.outcome
}

func (a *Admission) SetCookie() string {
	return a.setCookie
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
	known, err := a.store.Identity(ctx, provider, claim.Subject())
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

	current := visitor.account
	outcome, err := accounts.OutcomeOf(intent, current, known, owner, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to link %s: %w", provider, err)
	}
	var account accounts.AccountID
	switch outcome {
	case accounts.SignedIn:
		account = known.Account()
	case accounts.Linked:
		account = current.ID()
	case accounts.Joined:
		account = owner.ID()
	case accounts.Created:
		if account, err = a.ids.NewID(); err != nil {
			return nil, fmt.Errorf("failed to get an account id: %w", err)
		}
	}

	token, err := a.tokens.NewToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get a session token: %w", err)
	}
	session := accounts.LinkedSession(account, token, a.lifetime, now)
	signIn := accounts.NewSignIn(session).WithReplaced(visitor.replaces)
	if outcome == accounts.Created {
		signIn = signIn.WithNewAccount()
	}
	if outcome != accounts.SignedIn {
		signIn = signIn.WithIdentity(accounts.NewIdentity(provider, claim, account, now))
	}

	signedIn := &authv1.SignedIn{AccountId: account.String(), IdentityName: claim.Name()}
	if current != nil {
		signedIn.PreviousAccountId = current.ID().String()
	}
	err = a.transactioner.InTx(ctx, func(ctx context.Context) error {
		if err := a.store.SaveSignIn(ctx, signIn); err != nil {
			return fmt.Errorf("failed to save the sign-in: %w", err)
		}
		if outcome == accounts.Created {
			if err := a.arrivals.Record(ctx, account, visitor.landing, now); err != nil {
				return fmt.Errorf("failed to record the new account's arrival: %w", err)
			}
		}
		if err := a.outbox.Record(ctx, signedIn); err != nil {
			return fmt.Errorf("failed to record the sign-in: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Admission{account: account, outcome: outcome, setCookie: session.Cookie(token, now)}, nil
}
