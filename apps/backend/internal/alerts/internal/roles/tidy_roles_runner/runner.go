package tidy_roles_runner

import (
	"context"
	"log/slog"
)

type Book interface {
	Tidy(ctx context.Context) error
}

func New(book Book, logger *slog.Logger) Runner {
	return Runner{book: book, logger: logger}
}

// Once a boot: a color or a role the game changed reaches the server with the deploy that changed it.
type Runner struct {
	book   Book
	logger *slog.Logger
}

func (r Runner) Name() string { return "alerts-roles-tidy" }

func (r Runner) Run(ctx context.Context) {
	if err := r.book.Tidy(ctx); err != nil {
		r.logger.ErrorContext(ctx, "failed to tidy the Discord roles", slog.Any("error", err))
		return
	}
	r.logger.InfoContext(ctx, "Discord roles tidied")
}
