package rejected_subscriber

import (
	"context"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, submission string) error
}

func New(useCase UseCase) Subscriber {
	return Subscriber{useCase: useCase}
}

// Subscriber refunds what was paid for a flag the operator rejected.
type Subscriber struct {
	useCase UseCase
}

var _ cpredis.Handler[*flagsv1.SubmissionRejected] = Subscriber{}

func (s Subscriber) Handle(ctx context.Context, event *flagsv1.SubmissionRejected) error {
	return s.useCase.Execute(ctx, event.GetSubmissionId()) //nolint:wrapcheck // the use case named it.
}
