package log_set_rules

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/tempo/usecases/set_rules_usecase"
)

func New(inner set_rules_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  set_rules_usecase.Executor
	logger *slog.Logger
}

var _ set_rules_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, in set_rules_usecase.In) error {
	err := l.inner.Execute(ctx, in)

	attributes := []any{
		slog.Float64("refillMultiplier", in.RefillMultiplier),
		slog.Duration("boxInterval", in.BoxInterval),
		slog.String("gift", string(in.GiftTag)),
		slog.Time("giftMadeBefore", in.GiftMadeBefore),
		slog.Bool("frozen", in.Frozen),
	}
	if err != nil {
		l.logger.Warn("refused the rules", append(attributes, slog.Any("error", err))...)
	} else {
		l.logger.Info("rules set", attributes...)
	}

	return err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
