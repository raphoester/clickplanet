package prune_tokens_usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const (
	Interval = time.Hour
	timeout  = 30 * time.Second
)

type Tokens interface {
	DeleteLapsed(ctx context.Context, now time.Time) error
}

func New(tokens Tokens, clock cptime.Clock) *UseCase {
	return &UseCase{tokens: tokens, clock: clock}
}

type UseCase struct {
	tokens Tokens
	clock  cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context) error {
	if err := u.tokens.DeleteLapsed(ctx, u.clock.Now()); err != nil {
		return fmt.Errorf("failed to prune the admin tokens: %w", err)
	}
	return nil
}

func NewRunner(useCase *UseCase, logger *slog.Logger) *Runner {
	return &Runner{useCase: useCase, logger: logger}
}

// A lapsed token is refused anyway: the prune only keeps the table to the live ones, once at boot and every hour.
type Runner struct {
	useCase *UseCase
	logger  *slog.Logger
}

func (r *Runner) Name() string {
	return "admin-token-prune"
}

func (r *Runner) Run(ctx context.Context) {
	ticker := time.NewTicker(Interval)
	defer ticker.Stop()

	for {
		pruning, cancel := context.WithTimeout(ctx, timeout)
		if err := r.useCase.Execute(pruning); err != nil && ctx.Err() == nil {
			r.logger.Error("failed to prune the admin tokens", slog.Any("error", err))
		}
		cancel()

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
