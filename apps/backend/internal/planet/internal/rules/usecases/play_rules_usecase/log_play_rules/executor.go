package log_play_rules

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rules/usecases/play_rules_usecase"
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
	if took {
		written, _ := protojson.Marshal(taken.Rules)
		l.logger.Warn("rules in force", slog.String("set", string(taken.Name)), slog.String("rules", string(written)))
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to play the rules", slog.Any("error", err))
	}
	return taken, took, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
