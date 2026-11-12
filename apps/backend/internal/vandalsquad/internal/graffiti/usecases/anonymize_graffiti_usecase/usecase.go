package anonymize_graffiti_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
	"github.com/raphoester/clickplanet.lol-backend/internal/vandalsquad/internal/graffiti"
)

type Graffiti interface {
	PaintedBy(ctx context.Context, account cpsession.AccountID) ([]graffiti.Graffiti, error)
	Save(ctx context.Context, graffiti ...graffiti.Graffiti) error
}

func New(kept Graffiti) *UseCase {
	return &UseCase{graffiti: kept}
}

type UseCase struct {
	graffiti Graffiti
}

func (u *UseCase) Execute(ctx context.Context, account cpsession.AccountID) error {
	painted, err := u.graffiti.PaintedBy(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the graffiti the account painted: %w", err)
	}

	if err := u.graffiti.Save(ctx, graffiti.AllAnonymized(painted)...); err != nil {
		return fmt.Errorf("failed to anonymize the graffiti the account painted: %w", err)
	}
	return nil
}
