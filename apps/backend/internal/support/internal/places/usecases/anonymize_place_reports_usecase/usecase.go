package anonymize_place_reports_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
)

type Reports interface {
	FiledBy(ctx context.Context, reporter places.AccountID) ([]places.Report, error)
	Save(ctx context.Context, reports ...places.Report) error
}

func New(kept Reports) *UseCase {
	return &UseCase{reports: kept}
}

type UseCase struct {
	reports Reports
}

func (u *UseCase) Execute(ctx context.Context, account places.AccountID) error {
	filed, err := u.reports.FiledBy(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the places the account reported: %w", err)
	}
	if err := u.reports.Save(ctx, places.AllAnonymized(filed)...); err != nil {
		return fmt.Errorf("failed to anonymize the places the account reported: %w", err)
	}
	return nil
}
