package log_receive_notice

import (
	"context"
	"log/slog"

	"github.com/raphoester/clickplanet.lol-backend/internal/shop/internal/notices"
)

type Executor interface {
	Execute(ctx context.Context, notice notices.Notice) error
}

func New(inner Executor, logger *slog.Logger) Log {
	return Log{inner: inner, logger: logger}
}

// Log says what each Stripe event did: a payment is rare, and the operator answers for each.
type Log struct {
	inner  Executor
	logger *slog.Logger
}

func (l Log) Execute(ctx context.Context, notice notices.Notice) error {
	err := l.inner.Execute(ctx, notice)
	attributes := []any{slog.String("event", notice.Event.String()), slog.String("kind", string(notice.Kind))}
	switch notice.Kind {
	case notices.Paid:
		order := notice.Payment.Order
		attributes = append(attributes, slog.String("payment", notice.Payment.ID.String()),
			slog.String("account", order.Buyer.String()), slog.String("plan", string(order.Plan)),
			slog.String("submission", order.Target.Submission()), slog.String("flag", order.Target.Flag()),
			slog.Int64("amount", notice.Payment.Amount), slog.String("currency", notice.Payment.Currency))
	case notices.TakenBack:
		attributes = append(attributes, slog.String("intent", notice.Intent.String()))
	case notices.Renewal:
		attributes = append(attributes, slog.String("renewal", notice.Renewal.ID.String()),
			slog.String("state", string(notice.Renewal.State)), slog.Time("period_ends_at", notice.Renewal.PeriodEndsAt))
	case notices.Ignored:
		attributes = append(attributes, slog.String("why", notice.Why))
	}
	if err != nil {
		l.logger.ErrorContext(ctx, "failed to act on a Stripe event", append(attributes, slog.Any("error", err))...)
		return err //nolint:wrapcheck // a decorator adds nothing.
	}
	l.logger.InfoContext(ctx, "Stripe event heard", attributes...)
	return nil
}
