package audit_retell_missions_done

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/usecases/retell_missions_done_usecase"
)

func New(inner retell_missions_done_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  retell_missions_done_usecase.Executor
	logger *slog.Logger
}

var _ retell_missions_done_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context, in retell_missions_done_usecase.In) (retell_missions_done_usecase.Out, error) {
	out, err := a.inner.Execute(ctx, in)
	attrs := []any{slog.Bool("dry_run", in.DryRun), slog.Int("accounts", out.Accounts)}
	if err != nil {
		a.logger.Warn("admin missions done retold in part", append(attrs, slog.Any("error", err))...)
		return out, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	a.logger.Warn("admin missions done retold", attrs...)
	return out, nil
}
