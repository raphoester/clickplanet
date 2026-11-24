package grant_time_handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

const timeout = 5 * time.Second

type UseCase interface {
	Execute(ctx context.Context, paid grants.Paid) error
}

func New(useCase UseCase, logger *slog.Logger) Handler {
	return Handler{useCase: useCase, logger: logger}
}

// The command's edge: a payment's time, granted once.
type Handler struct {
	useCase UseCase
	logger  *slog.Logger
}

var _ cpredis.Handler[*flagsv1.GrantTime] = Handler{}

func (h Handler) Handle(ctx context.Context, command *flagsv1.GrantTime) error {
	period, err := periodOf(command.GetPeriod())
	if err != nil {
		return err
	}
	paid, err := grants.ParsePaid(command.GetGrantId(), command.GetTarget().GetSubmissionId(),
		command.GetTarget().GetFlag(), period, command.GetPaidAt().AsTime())
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = h.useCase.Execute(ctx, paid)
	attributes := []any{
		slog.String("grant", paid.Grant.String()), slog.String("submission", command.GetTarget().GetSubmissionId()),
		slog.String("flag", command.GetTarget().GetFlag()), slog.String("period", string(period)),
	}
	switch {
	case err == nil:
		h.logger.Info("time paid for granted", attributes...)
		return nil
	// Paid for, and no try again grants it: the operator refunds it.
	case errors.Is(err, grants.ErrRefused):
		h.logger.Warn("a payment granted no time: the shop refunds it", append(attributes, slog.Any("error", err))...)
		return nil
	default:
		return err //nolint:wrapcheck // the use case named it.
	}
}

func periodOf(period flagsv1.TimePeriod) (grants.Period, error) {
	switch period {
	case flagsv1.TimePeriod_TIME_PERIOD_WEEK:
		return grants.Week, nil
	case flagsv1.TimePeriod_TIME_PERIOD_MONTH:
		return grants.Month, nil
	case flagsv1.TimePeriod_TIME_PERIOD_YEAR:
		return grants.Year, nil
	case flagsv1.TimePeriod_TIME_PERIOD_UNSPECIFIED:
	}
	return "", fmt.Errorf("%w: %v", grants.ErrInvalidPeriod, period)
}
