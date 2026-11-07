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
	SaveOnce(ctx context.Context, decision decisions.Decision) (bool, error)
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
	// A command's: a report decided already keeps its decision, the operator's or this command's told again, and
	// the call is ErrDecidedAlready. The operator's call replaces it.
	Once bool
}

func (u *UseCase) Execute(ctx context.Context, in In) (time.Time, error) {
	decision, err := decisions.NewDecision(in.Report, in.Outcome, in.Comment, u.clock.Now())
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to decide the report: %w", err)
	}

	if !in.Once {
		if err := u.saver.Save(ctx, decision); err != nil {
			return time.Time{}, fmt.Errorf("failed to keep the decision: %w", err)
		}
		return decision.TakeSnapshot().At, nil
	}

	kept, err := u.saver.SaveOnce(ctx, decision)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to keep the decision: %w", err)
	}
	if !kept {
		return time.Time{}, decisions.ErrDecidedAlready
	}
	return decision.TakeSnapshot().At, nil
}
