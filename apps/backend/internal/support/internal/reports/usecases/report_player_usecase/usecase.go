package report_player_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

type Reports interface {
	FiledBy(ctx context.Context, reporter reports.AccountID, about reports.AccountID) ([]reports.Report, error)
	Save(ctx context.Context, reports ...reports.Report) error
}

type In struct {
	Reporter  reports.AccountID
	Reported  reports.AccountID
	Complaint reports.Complaint
}

const timeout = 5 * time.Second

func New(ids reports.IDProvider, clock cptime.Clock, kept Reports) *UseCase {
	return &UseCase{ids: ids, clock: clock, reports: kept}
}

type UseCase struct {
	ids     reports.IDProvider
	clock   cptime.Clock
	reports Reports
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	earlier, err := u.reports.FiledBy(ctx, in.Reporter, in.Reported)
	if err != nil {
		return fmt.Errorf("failed to read what the reporter already filed about the player: %w", err)
	}

	id, err := u.ids.NewID()
	if err != nil {
		return fmt.Errorf("failed to draw a report id: %w", err)
	}

	report, err := reports.NewReport(id, in.Reporter, in.Reported, in.Complaint, u.clock.Now(), earlier)
	if err != nil {
		return fmt.Errorf("cannot take the report: %w", err)
	}

	if err := u.reports.Save(ctx, report); err != nil {
		return fmt.Errorf("failed to keep the report: %w", err)
	}
	return nil
}
