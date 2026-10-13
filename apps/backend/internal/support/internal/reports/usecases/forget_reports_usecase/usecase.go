package forget_reports_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

type Reports interface {
	Naming(ctx context.Context, account reports.AccountID) ([]reports.Report, error)
	Save(ctx context.Context, reports ...reports.Report) error
}

func New(kept Reports) *UseCase {
	return &UseCase{reports: kept}
}

type UseCase struct {
	reports Reports
}

func (u *UseCase) Execute(ctx context.Context, account reports.AccountID) error {
	naming, err := u.reports.Naming(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the reports that name the account: %w", err)
	}

	if err := u.reports.Save(ctx, reports.AllForgotten(naming)...); err != nil {
		return fmt.Errorf("failed to forget the reports that name the account: %w", err)
	}
	return nil
}
