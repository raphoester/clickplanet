package approved_subscriber

import (
	"context"
	"errors"
	"log/slog"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, submission string) error
}

func New(useCase UseCase, logger *slog.Logger) Subscriber {
	return Subscriber{useCase: useCase, logger: logger}
}

// Subscriber opens an approved submission's flag with the time paid while it waited.
type Subscriber struct {
	useCase UseCase
	logger  *slog.Logger
}

var _ cpredis.Handler[*flagsv1.SubmissionApproved] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *flagsv1.SubmissionApproved) error {
	err := s.useCase.Execute(ctx, event.GetSubmissionId())
	switch {
	case err == nil:
		return nil
	case errors.Is(err, submissions.ErrUnknownSubmission), errors.Is(err, submissions.ErrNotApproved),
		errors.Is(err, catalog.ErrNameTaken), errors.Is(err, catalog.ErrEnded), errors.Is(err, grants.ErrNotTheFlag),
		errors.Is(err, grants.ErrStillWaiting):
		s.logger.Error("time paid for an approved flag did not open it: open it by hand or refund it",
			slog.String("submission", event.GetSubmissionId()), slog.Any("error", err))
		return nil
	default:
		return err //nolint:wrapcheck // the use case named it.
	}
}
