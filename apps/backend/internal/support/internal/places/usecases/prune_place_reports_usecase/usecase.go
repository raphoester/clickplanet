package prune_place_reports_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/places"
)

type Reports interface {
	WithReporterBefore(ctx context.Context, cutoff time.Time, limit int) ([]places.Report, error)
	Save(ctx context.Context, reports ...places.Report) error
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

	var anonymized int64
	for {
		old, err := u.reports.WithReporterBefore(ctx, cutoff, batch)
		if err != nil {
			return anonymized, fmt.Errorf("failed to read the place reports past retention: %w", err)
		}
		if len(old) == 0 {
			return anonymized, nil
		}

		if err := u.reports.Save(ctx, places.AllAnonymized(old)...); err != nil {
			return anonymized, fmt.Errorf("failed to anonymize the place reports past retention: %w", err)
		}
		anonymized += int64(len(old))
	}
}
