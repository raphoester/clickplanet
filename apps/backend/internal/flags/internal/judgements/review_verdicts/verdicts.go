// Package review_verdicts applies the judge's rulings through the operator's own use cases.
package review_verdicts

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/approve_submission_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions/usecases/reject_submission_usecase"
)

type Approver interface {
	Execute(ctx context.Context, in approve_submission_usecase.In) (submissions.Snapshot, error)
}

type Rejecter interface {
	Execute(ctx context.Context, in reject_submission_usecase.In) (submissions.Snapshot, error)
}

func New(approve Approver, reject Rejecter) Verdicts {
	return Verdicts{approve: approve, reject: reject}
}

type Verdicts struct {
	approve Approver
	reject  Rejecter
}

func (v Verdicts) Approve(ctx context.Context, id submissions.SubmissionID) error {
	_, err := v.approve.Execute(ctx, approve_submission_usecase.In{Submission: id.String()})
	return err //nolint:wrapcheck // the use case names it.
}

func (v Verdicts) Reject(ctx context.Context, id submissions.SubmissionID, rejection submissions.Rejection) error {
	_, err := v.reject.Execute(ctx, reject_submission_usecase.In{
		Submission: id.String(), Reason: rejection.Reason, Note: rejection.Note,
	})
	return err //nolint:wrapcheck // the use case names it.
}
