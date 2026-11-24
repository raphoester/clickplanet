package reject_submission_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type In struct {
	Submission string
	Reason     submissions.Reason
	Note       string
}

func New(clock cptime.Clock, store submissions.Store) *UseCase {
	return &UseCase{clock: clock, store: store}
}

type UseCase struct {
	clock cptime.Clock
	store submissions.Store
}

func (u *UseCase) Execute(ctx context.Context, in In) (submissions.Snapshot, error) {
	id, err := submissions.ParseSubmissionID(in.Submission)
	if err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}
	rejection, err := submissions.ParseRejection(in.Reason, in.Note)
	if err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}
	submission, err := u.store.Submission(ctx, id)
	if err != nil {
		return submissions.Snapshot{}, fmt.Errorf("failed to read the submission: %w", err)
	}
	rejected, err := submission.Rejected(rejection, u.clock.Now())
	if err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the rule names it.
	}
	if err := u.store.Save(ctx, rejected); err != nil {
		return submissions.Snapshot{}, fmt.Errorf("failed to keep the rejection: %w", err)
	}
	return rejected.TakeSnapshot(), nil
}
