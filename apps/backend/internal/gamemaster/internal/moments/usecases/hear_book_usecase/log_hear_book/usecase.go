package log_hear_book

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/usecases/hear_book_usecase"
)

func New(inner hear_book_usecase.UseCase, logger *slog.Logger) *Logged {
	return &Logged{inner: inner, logger: logger}
}

type Logged struct {
	inner  hear_book_usecase.UseCase
	logger *slog.Logger
}

var _ hear_book_usecase.UseCase = (*Logged)(nil)

func (l *Logged) Execute(ctx context.Context, closed moments.ClosedBook) (hear_book_usecase.Heard, error) {
	heard, err := l.inner.Execute(ctx, closed)
	attrs := []any{slog.String("book", string(closed.Book))}
	switch {
	case err != nil:
		l.logger.Error("failed to hear a book scoring closed", append(attrs, slog.Any("error", err))...)
	case heard == hear_book_usecase.Told:
		l.logger.Info("scoring closed a book", attrs...)
	case heard == hear_book_usecase.Unknown:
		l.logger.Warn("scoring closed a book no season has: dropped", attrs...)
	}
	return heard, err //nolint:wrapcheck // a decorator adds a log line, not a sentence.
}
