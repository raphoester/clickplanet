package admit_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/admingate/admission"
)

type Operators interface {
	Operator(ctx context.Context, cookie string) (admission.Operator, error)
	Connection(ctx context.Context, token string) (admission.Caller, error)
}

func New(page admission.Page, operators Operators) *UseCase {
	return &UseCase{page: page, operators: operators}
}

type UseCase struct {
	page      admission.Page
	operators Operators
}

func (u *UseCase) Execute(ctx context.Context, visit admission.Visit) (admission.Caller, error) {
	if visit.Token != "" {
		return u.keyed(ctx, visit)
	}

	if err := u.page.OriginError(visit.Origin); err != nil {
		return admission.Caller{}, fmt.Errorf("failed to admit the caller: %w", err)
	}
	if u.page.SignIn(visit.Procedure) {
		return admission.Caller{}, nil
	}

	operator, err := u.operators.Operator(ctx, visit.Cookie)
	if err != nil {
		return admission.Caller{}, fmt.Errorf("failed to admit the caller: %w", err)
	}

	return admission.SessionOf(operator), nil
}

// A browser never sends a bearer token on its own, so a token call skips the origin check.
func (u *UseCase) keyed(ctx context.Context, visit admission.Visit) (admission.Caller, error) {
	if err := u.page.TokenError(visit.Procedure); err != nil {
		return admission.Caller{}, fmt.Errorf("failed to admit the token: %w", err)
	}

	caller, err := u.operators.Connection(ctx, visit.Token)
	if err != nil {
		return admission.Caller{}, fmt.Errorf("failed to admit the token: %w", err)
	}

	return caller, nil
}
