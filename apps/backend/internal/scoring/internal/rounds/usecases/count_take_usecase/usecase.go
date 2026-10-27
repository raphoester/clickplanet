package count_take_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
)

type Takes interface {
	Count(ctx context.Context, take rounds.Take) error
}

func New(takes Takes) *UseCase {
	return &UseCase{takes: takes}
}

// UseCase counts each take in the rounds of their own whose window holds it, for their numbers.
type UseCase struct {
	takes Takes
}

func (u *UseCase) Execute(ctx context.Context, take rounds.Take) error {
	if err := u.takes.Count(ctx, take); err != nil {
		return fmt.Errorf("failed to count the take: %w", err)
	}
	return nil
}
