package log_play_rules

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/usecases/play_rules_usecase"
)

func New(inner play_rules_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  play_rules_usecase.Executor
	logger *slog.Logger
}

var _ play_rules_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (rules.Snapshot, bool, error) {
	taken, took, err := l.inner.Execute(ctx)
	l.log(ctx, taken, took, err)
	return taken, took, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}

func (l *Logged) Play(ctx context.Context, choice rules.Choice) (rules.Snapshot, bool, error) {
	taken, took, err := l.inner.Play(ctx, choice)
	l.log(ctx, taken, took, err)
	return taken, took, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}

func (l *Logged) log(ctx context.Context, taken rules.Snapshot, took bool, err error) {
	if took {
		written, _ := protojson.Marshal(taken.Rules)
		l.logger.Warn("antibot rules in force",
			slog.String("set", string(taken.Name)),
			slog.String("rules", string(written)))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to play the antibot rules", slog.Any("error", err))
	}
}
