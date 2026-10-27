package log_play_rules

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rules/usecases/play_rules_usecase"
)

func New(inner play_rules_usecase.Executor, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  play_rules_usecase.Executor
	logger *slog.Logger
}

var _ play_rules_usecase.Executor = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context) (play_rules_usecase.Played, bool, error) {
	played, took, err := l.inner.Execute(ctx)
	l.log(ctx, played, took, err)
	return played, took, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}

func (l *Logged) Play(ctx context.Context, choice rules.Choice) (play_rules_usecase.Played, bool, error) {
	played, took, err := l.inner.Play(ctx, choice)
	l.log(ctx, played, took, err)
	return played, took, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}

func (l *Logged) log(ctx context.Context, played play_rules_usecase.Played, took bool, err error) {
	if took {
		written, _ := protojson.Marshal(played.Set.Rules)
		attrs := []any{slog.String("set", string(played.Set.Name)), slog.String("rules", string(written))}
		if !played.Window.None() {
			attrs = append(attrs, slog.Time("startsAt", played.Window.StartsAt), slog.Time("endsAt", played.Window.EndsAt))
		}
		l.logger.Warn("season rules in force", attrs...)
	}
	if err != nil && ctx.Err() == nil {
		l.logger.Error("failed to play the season rules", slog.Any("error", err))
	}
}
