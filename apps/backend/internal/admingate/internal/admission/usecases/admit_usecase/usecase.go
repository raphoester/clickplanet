package admit_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/admingate/internal/admission"
)

type Operators interface {
	Operator(ctx context.Context, cookie string) (admission.Operator, error)
}

func New(page admission.Page, operators Operators) *UseCase {
	return &UseCase{page: page, operators: operators}
}

type UseCase struct {
	page      admission.Page
	operators Operators
}

func (u *UseCase) Execute(ctx context.Context, visit admission.Visit) (admission.Operator, error) {
	if err := u.page.OriginError(visit.Origin); err != nil {
		return "", fmt.Errorf("failed to admit the caller: %w", err)
	}

	operator, err := u.operators.Operator(ctx, visit.Cookie)
	if err != nil {
		return "", fmt.Errorf("failed to admit the caller: %w", err)
	}

	return operator, nil
}
