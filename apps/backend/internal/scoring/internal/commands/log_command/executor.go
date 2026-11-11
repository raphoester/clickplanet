package log_command

import (
	"context"
	"errors"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/books"
)

type Executor[C, A any] interface {
	Execute(ctx context.Context, command C) (A, error)
}

func New[C, A any](name string, inner Executor[C, A], logger *slog.Logger) *Logged[C, A] {
	return &Logged[C, A]{name: name, inner: inner, logger: logger}
}

// Logged logs each answer and each refusal, and leaves any other failure to its caller's log.
type Logged[C, A any] struct {
	name   string
	inner  Executor[C, A]
	logger *slog.Logger
}

func (l *Logged[C, A]) Execute(ctx context.Context, command C) (A, error) {
	answer, err := l.inner.Execute(ctx, command)
	if err == nil {
		l.logger.Info("scoring command answered",
			slog.String("command", l.name), slog.Any("args", command), slog.Any("answer", answer))
	} else if errors.Is(err, books.ErrRefused) {
		l.logger.Warn("scoring command refused",
			slog.String("command", l.name), slog.Any("args", command), slog.Any("error", err))
	}
	return answer, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
