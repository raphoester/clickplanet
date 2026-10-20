package decide_report_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
)

type Saver[R comparable] interface {
	Save(ctx context.Context, decision decisions.Decision[R]) error
}

func New[R comparable](saver Saver[R], clock cptime.Clock) *UseCase[R] {
	return &UseCase[R]{saver: saver, clock: clock}
}

type UseCase[R comparable] struct {
	saver Saver[R]
	clock cptime.Clock
}

type In[R comparable] struct {
	Report  R
	Outcome decisions.Outcome
	Comment string
}

func (u *UseCase[R]) Execute(ctx context.Context, in In[R]) (time.Time, error) {
	decision, err := decisions.NewDecision(in.Report, in.Outcome, in.Comment, u.clock.Now())
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to decide the report: %w", err)
	}

	if err := u.saver.Save(ctx, decision); err != nil {
		return time.Time{}, fmt.Errorf("failed to keep the decision: %w", err)
	}
	return decision.TakeSnapshot().At, nil
}
