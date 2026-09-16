// Package fail_open_accounts keeps a mint going when the auth module fails: the
// caller gets a token with no account, and plays as a caller with none did before
// accounts existed. A broken auth module must not stop the clicks.
package fail_open_accounts

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/raphoester/clickplanet.lol-backend/internal/session/internal/domain"
)

type Accounts struct {
	next     domain.Accounts
	logger   *slog.Logger
	outcomes *prometheus.CounterVec
}

var _ domain.Accounts = (*Accounts)(nil)

func New(next domain.Accounts, logger *slog.Logger, registerer prometheus.Registerer) *Accounts {
	return &Accounts{
		next:   next,
		logger: logger,
		outcomes: promauto.With(registerer).NewCounterVec(prometheus.CounterOpts{
			Name: "session_account_resolutions_total",
			Help: "Account resolutions at mint, by outcome: account, none, or failed (minted with no account)",
		}, []string{"outcome"}),
	}
}

func (a *Accounts) Resolve(ctx context.Context, cookieHeader string, create bool) (*domain.Resolution, error) {
	resolution, err := a.next.Resolve(ctx, cookieHeader, create)
	switch {
	case err == nil:
		a.outcomes.WithLabelValues("account").Inc()
		return resolution, nil
	case errors.Is(err, domain.ErrNoAccount):
		a.outcomes.WithLabelValues("none").Inc()
		return nil, fmt.Errorf("failed to resolve the account: %w", err)
	default:
		a.outcomes.WithLabelValues("failed").Inc()
		a.logger.Warn("minted a session with no account", slog.Any("error", err))
		return nil, fmt.Errorf("%w: the auth module failed: %w", domain.ErrNoAccount, err)
	}
}
