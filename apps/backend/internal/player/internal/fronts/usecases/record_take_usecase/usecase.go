package record_take_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts"
)

type Fronts interface {
	RecordTake(ctx context.Context, take fronts.Take) error
}

type UseCase struct {
	fronts Fronts
}

func New(fronts Fronts) *UseCase {
	return &UseCase{fronts: fronts}
}

func (u *UseCase) Execute(ctx context.Context, take fronts.Take) error {
	if err := u.fronts.RecordTake(ctx, take); err != nil {
		return fmt.Errorf("failed to count the take for and against its countries: %w", err)
	}
	return nil
}
