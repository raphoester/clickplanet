package report_player_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

type Accounts interface {
	AccountOf(ctx context.Context, name reports.ShownName) (reports.AccountID, error)
}

type Reports interface {
	FiledBy(ctx context.Context, reporter reports.AccountID, about reports.AccountID) ([]reports.Report, error)
	Save(ctx context.Context, reports ...reports.Report) error
}

type In struct {
	Reporter  reports.AccountID
	Name      string
	Complaint reports.Complaint
}

const timeout = 5 * time.Second

func New(accounts Accounts, ids reports.IDProvider, clock cptime.Clock, kept Reports) *UseCase {
	return &UseCase{accounts: accounts, ids: ids, clock: clock, reports: kept}
}

type UseCase struct {
	accounts Accounts
	ids      reports.IDProvider
	clock    cptime.Clock
	reports  Reports
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	name, err := reports.ShownNameOf(in.Name)
	if err != nil {
		return fmt.Errorf("cannot take the report: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	reported, err := u.accounts.AccountOf(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to find the player reported: %w", err)
	}

	earlier, err := u.reports.FiledBy(ctx, in.Reporter, reported)
	if err != nil {
		return fmt.Errorf("failed to read what the reporter already filed about the player: %w", err)
	}

	id, err := u.ids.NewID()
	if err != nil {
		return fmt.Errorf("failed to draw a report id: %w", err)
	}

	report, err := reports.NewReport(id, in.Reporter, reported, in.Complaint, u.clock.Now(), earlier)
	if err != nil {
		return fmt.Errorf("cannot take the report: %w", err)
	}

	if err := u.reports.Save(ctx, report); err != nil {
		return fmt.Errorf("failed to keep the report: %w", err)
	}
	return nil
}
