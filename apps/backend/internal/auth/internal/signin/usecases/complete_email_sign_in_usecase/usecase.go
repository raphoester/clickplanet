package complete_email_sign_in_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/signin"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type In struct {
	Code         string
	CookieHeader string
}

type Out = signin.Admission

type UseCase struct {
	challenges *signin.Challenges
	admitter   *signin.Admitter
	clock      cptime.Clock
}

func New(challenges *signin.Challenges, admitter *signin.Admitter, clock cptime.Clock) *UseCase {
	return &UseCase{challenges: challenges, admitter: admitter, clock: clock}
}

func (u *UseCase) Execute(ctx context.Context, in In) (*Out, error) {
	if u.challenges.Off() {
		return nil, signin.ErrSignInOff
	}
	challenge, err := u.challenges.Opened(in.CookieHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to open the challenge: %w", err)
	}
	now := u.clock.Now()
	if err := u.challenges.Guess(challenge, in.Code, now); err != nil {
		return nil, fmt.Errorf("failed to check the code: %w", err)
	}

	visitor, err := u.admitter.Visitor(ctx, in.CookieHeader, now)
	if err != nil {
		return nil, fmt.Errorf("failed to find the browser's account: %w", err)
	}
	if err := challenge.AccountError(visitor.Account()); err != nil {
		return nil, fmt.Errorf("failed to check the account: %w", err)
	}

	admission, err := u.admitter.Admit(ctx, signin.Email, challenge.Intent(), challenge.Claim(), visitor, now)
	if err != nil {
		return nil, fmt.Errorf("failed to sign in: %w", err)
	}
	return admission, nil
}
