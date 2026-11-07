package decide_report_handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	supportv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/support/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/decisions/usecases/decide_report_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/reports"
	"github.com/raphoester/clickplanet.lol-backend/internal/support/internal/subscribers"
)

var ErrUnknownOutcome = errors.New("not an outcome of a report")

type UseCase interface {
	Execute(ctx context.Context, in decide_report_usecase.In) (time.Time, error)
}

func New(useCase UseCase, logger *slog.Logger) Handler {
	return Handler{useCase: useCase, logger: logger}
}

// The command's edge, as the admin handler is the call's: it decides once.
type Handler struct {
	useCase UseCase
	logger  *slog.Logger
}

var _ cpredis.Handler[*supportv1.DecideReport] = Handler{}

func (h Handler) Handle(ctx context.Context, command *supportv1.DecideReport) error {
	report, err := reports.ReportIDOf(command.GetReportId())
	if err != nil {
		return err //nolint:wrapcheck // the sentinel says what is wrong.
	}
	outcome, err := outcomeOf(command.GetOutcome())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, subscribers.Timeout)
	defer cancel()

	_, err = h.useCase.Execute(ctx, decide_report_usecase.In{
		Report: report, Outcome: outcome, Comment: command.GetComment(), Once: true,
	})
	attributes := []any{
		slog.String("report", uuid.UUID(report).String()), slog.String("outcome", string(outcome)),
		slog.String("comment", command.GetComment()),
	}
	switch {
	case err == nil:
		h.logger.Info("report decided by a command", attributes...)
		return nil
	// Told again, or the operator was first: the decision kept stays.
	case errors.Is(err, decisions.ErrDecidedAlready):
		return nil
	// Forgotten with an account, or past the retention: nothing to close, and nothing a try again changes.
	case errors.Is(err, decisions.ErrUnknownReport):
		h.logger.Info("a command decided a report no longer kept", attributes...)
		return nil
	default:
		return err //nolint:wrapcheck // the use case named it.
	}
}

func outcomeOf(outcome supportv1.ReportOutcome) (decisions.Outcome, error) {
	switch outcome {
	case supportv1.ReportOutcome_REPORT_OUTCOME_ACTED:
		return decisions.OutcomeActed, nil
	case supportv1.ReportOutcome_REPORT_OUTCOME_DISMISSED:
		return decisions.OutcomeDismissed, nil
	case supportv1.ReportOutcome_REPORT_OUTCOME_UNSPECIFIED:
	}
	return "", fmt.Errorf("%w: %v", ErrUnknownOutcome, outcome)
}
