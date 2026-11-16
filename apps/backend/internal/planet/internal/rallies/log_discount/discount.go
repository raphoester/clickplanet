package log_discount

import (
	"context"
	"log/slog"
)

type Discount interface {
	Cost(ctx context.Context, flag string, tile uint32) (float64, error)
}

func New(discount Discount, logger *slog.Logger) Logged {
	return Logged{discount: discount, logger: logger}
}

// Logged warns of a read of the rallies that failed: the click it priced cost a whole one.
type Logged struct {
	discount Discount
	logger   *slog.Logger
}

func (l Logged) Cost(ctx context.Context, flag string, tile uint32) (float64, error) {
	cost, err := l.discount.Cost(ctx, flag, tile)
	if err != nil {
		l.logger.WarnContext(ctx, "a click priced whole: the rallies could not be read", slog.Any("error", err))
	}
	return cost, err //nolint:wrapcheck // a decorator.
}
