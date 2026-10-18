package decide_report_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
)

type Saver interface {
	Save(ctx context.Context, decision decisions.Decision) error
}

func New(saver Saver, clock cptime.Clock) *UseCase {
	return &UseCase{saver: saver, clock: clock}
}

type UseCase struct {
	saver Saver
	clock cptime.Clock
}

type In struct {
	Report  reports.ReportID
	Outcome decisions.Outcome
	Comment string
}

func (u *UseCase) Execute(ctx context.Context, in In) (time.Time, error) {
	decision, err := decisions.NewDecision(in.Report, in.Outcome, in.Comment, u.clock.Now())
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to decide the report: %w", err)
	}

	if err := u.saver.Save(ctx, decision); err != nil {
		return time.Time{}, fmt.Errorf("failed to keep the decision: %w", err)
	}
	return decision.TakeSnapshot().At, nil
}
