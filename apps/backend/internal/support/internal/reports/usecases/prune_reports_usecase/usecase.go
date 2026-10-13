package prune_reports_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

type Reports interface {
	FiledBefore(ctx context.Context, cutoff time.Time, limit int) ([]reports.Report, error)
	Save(ctx context.Context, reports ...reports.Report) error
}

const (
	timeout = 30 * time.Second
	batch   = 500
)

func New(retention time.Duration, clock cptime.Clock, kept Reports) *UseCase {
	return &UseCase{retention: retention, clock: clock, reports: kept}
}

type UseCase struct {
	retention time.Duration
	clock     cptime.Clock
	reports   Reports
}

func (u *UseCase) Execute(ctx context.Context) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cutoff := u.clock.Now().Add(-u.retention)

	var forgotten int64
	for {
		old, err := u.reports.FiledBefore(ctx, cutoff, batch)
		if err != nil {
			return forgotten, fmt.Errorf("failed to read the reports past retention: %w", err)
		}
		if len(old) == 0 {
			return forgotten, nil
		}

		if err := u.reports.Save(ctx, reports.AllForgotten(old)...); err != nil {
			return forgotten, fmt.Errorf("failed to forget the reports past retention: %w", err)
		}
		forgotten += int64(len(old))
	}
}
