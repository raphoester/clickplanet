// Package complete_email_sign_in_usecase finishes an email sign-in: it checks the code against the challenge, and signs the browser in to the address's account.
package complete_email_sign_in_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/accounts"
	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Limiter interface {
	Take(key string) (bool, cpratelimit.State)
}

type In struct {
	Code         string
	CookieHeader string
}

type Out = signin.Admission

type UseCase struct {
	offered  bool
	sealer   signin.ChallengeSealer
	attempts Limiter
	admitter *signin.Admitter
	clock    cptime.Clock
}

func New(
	offered bool,
	sealer signin.ChallengeSealer,
	attempts Limiter,
	store signin.IdentityStore,
	ids accounts.IDProvider,
	tokens accounts.TokenGenerator,
	lifetime accounts.Lifetime,
	events signin.Publisher,
	clock cptime.Clock,
) *UseCase {
	return &UseCase{
		offered:  offered,
		sealer:   sealer,
		attempts: attempts,
		admitter: signin.NewAdmitter(store, ids, tokens, lifetime, events),
		clock:    clock,
	}
}

// Execute answers signin.ErrSignInOff, signin.ErrFlowInvalid for a challenge to start again, and signin.ErrWrongCode for one to try
// again, and accounts.ErrIdentityLinkedElsewhere or accounts.ErrProviderAlreadyLinked for a link it refuses, having written nothing.
func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	if !u.offered {
		return nil, signin.ErrSignInOff
	}

	sealed, found := accounts.CookieValue(in.CookieHeader, signin.ChallengeCookieName)
	if !found {
		return nil, fmt.Errorf("%w: the browser sent no %s cookie", signin.ErrFlowInvalid, signin.ChallengeCookieName)
	}
	challenge, err := u.sealer.OpenedChallenge(sealed)
	if err != nil {
		return nil, fmt.Errorf("failed to open the challenge: %w", err)
	}

	// Every guess spends one, the right one too: the budget is what makes six digits enough.
	if allowed, _ := u.attempts.Take(challenge.ID); !allowed {
		return nil, fmt.Errorf("%w: too many codes were wrong", signin.ErrFlowInvalid)
	}
	now := u.clock.Now()
	if err := challenge.CodeError(in.Code, now); err != nil {
		return nil, fmt.Errorf("failed to check the code: %w", err)
	}

	visitor, err := u.admitter.Visitor(ctx, in.CookieHeader, now)
	if err != nil {
		return nil, fmt.Errorf("failed to find the browser's account: %w", err)
	}
	if err := challenge.AccountError(visitor.Account); err != nil {
		return nil, fmt.Errorf("failed to check the account: %w", err)
	}

	admission, err := u.admitter.Admit(ctx, signin.Email, challenge.Intent, challenge.Claim(), visitor, now)
	if err != nil {
		return nil, fmt.Errorf("failed to sign in: %w", err)
	}
	return admission, nil
}
