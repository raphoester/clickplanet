package verdict_subscriber

import (
	"context"
	"time"

	flagsv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/flags/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpredis"
)

type UseCase interface {
	Execute(ctx context.Context, submission string, decidedAt time.Time) error
}

func NewApproved(useCase UseCase) Approved {
	return Approved{useCase: useCase}
}

// Approved answers a submission's appeal with its approval.
type Approved struct {
	useCase UseCase
}

var _ cpredis.Handler[*flagsv1.SubmissionApproved] = Approved{}

func (s Approved) Handle(ctx context.Context, event *flagsv1.SubmissionApproved) error {
	return s.useCase.Execute(ctx, event.GetSubmissionId(), event.GetApprovedAt().AsTime()) //nolint:wrapcheck // the use case named it.
}

func NewRejected(useCase UseCase) Rejected {
	return Rejected{useCase: useCase}
}

// Rejected answers a submission's appeal with its rejection.
type Rejected struct {
	useCase UseCase
}

var _ cpredis.Handler[*flagsv1.SubmissionRejected] = Rejected{}

func (s Rejected) Handle(ctx context.Context, event *flagsv1.SubmissionRejected) error {
	return s.useCase.Execute(ctx, event.GetSubmissionId(), event.GetRejectedAt().AsTime()) //nolint:wrapcheck // the use case named it.
}
