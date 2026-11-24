package reject_submission_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type In struct {
	Submission string
	Reason     submissions.Reason
	Note       string
}

func New(clock cptime.Clock, db cpatomic.Transactioner, store submissions.Store, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{clock: clock, db: db, store: store, outbox: outbox}
}

type UseCase struct {
	clock  cptime.Clock
	db     cpatomic.Transactioner
	store  submissions.Store
	outbox cpatomic.Outbox
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
	var rejected submissions.Submission
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		submission, err := u.store.Submission(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to read the submission: %w", err)
		}
		if rejected, err = submission.Rejected(rejection, u.clock.Now()); err != nil {
			return err //nolint:wrapcheck // the rule names it.
		}
		if err := u.store.Save(ctx, rejected); err != nil {
			return fmt.Errorf("failed to keep the rejection: %w", err)
		}
		if err := u.outbox.Record(ctx, rejected.Facts()...); err != nil {
			return fmt.Errorf("failed to record the rejection: %w", err)
		}
		return nil
	}); err != nil {
		return submissions.Snapshot{}, err //nolint:wrapcheck // the transaction named it.
	}
	return rejected.TakeSnapshot(), nil
}
