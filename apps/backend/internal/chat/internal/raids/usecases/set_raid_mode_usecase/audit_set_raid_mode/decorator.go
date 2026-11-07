package audit_set_raid_mode

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids"
	"github.com/raphoester/clickplanet.lol-backend/internal/chat/internal/raids/usecases/set_raid_mode_usecase"
)

func New(inner set_raid_mode_usecase.Executor, logger *slog.Logger) *Audited {
	return &Audited{inner: inner, logger: logger}
}

type Audited struct {
	inner  set_raid_mode_usecase.Executor
	logger *slog.Logger
}

var _ set_raid_mode_usecase.Executor = (*Audited)(nil)

func (a *Audited) Execute(ctx context.Context, in set_raid_mode_usecase.In) (raids.RaidMode, bool, error) {
	mode, running, err := a.inner.Execute(ctx, in)

	attrs := []any{
		slog.Duration("duration", in.Duration), slog.String("reason", in.Reason), slog.Bool("running", running),
		slog.Time("endsAt", mode.TakeSnapshot().EndsAt),
		slog.String("operator", in.By.Account().String()), slog.String("client", in.By.Client()),
	}
	if err != nil {
		a.logger.Warn("admin chat raid mode failed", append(attrs, slog.Any("error", err))...)
		return mode, running, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
	}
	a.logger.Warn("admin chat raid mode", attrs...)
	return mode, running, nil
}
