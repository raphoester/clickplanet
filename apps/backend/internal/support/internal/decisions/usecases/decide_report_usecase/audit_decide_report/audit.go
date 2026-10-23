package audit_decide_report

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions/usecases/decide_report_usecase"
)

type Report interface {
	comparable
	fmt.Stringer
}

type UseCase[R Report] interface {
	Execute(ctx context.Context, in decide_report_usecase.In[R]) (time.Time, error)
}

// kind names the report decided in the log line: a player's or a place's.
func New[R Report](inner UseCase[R], kind string, logger *slog.Logger) *Audited[R] {
	return &Audited[R]{inner: inner, kind: kind, logger: logger}
}

type Audited[R Report] struct {
	inner  UseCase[R]
	kind   string
	logger *slog.Logger
}

func (a *Audited[R]) Execute(ctx context.Context, in decide_report_usecase.In[R]) (time.Time, error) {
	attributes := []any{
		slog.String("kind", a.kind),
		slog.String("report", in.Report.String()),
		slog.String("outcome", string(in.Outcome)),
		slog.String("comment", in.Comment),
	}

	decidedAt, err := a.inner.Execute(ctx, in)
	if err != nil {
		a.logger.Warn("admin report decision failed", append(attributes, slog.Any("error", err))...)
		return decidedAt, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}

	a.logger.Warn("admin report decision", attributes...)
	return decidedAt, nil
}
