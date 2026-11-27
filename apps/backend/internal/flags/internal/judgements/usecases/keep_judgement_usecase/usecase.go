package keep_judgement_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Submissions interface {
	Submission(ctx context.Context, id submissions.SubmissionID) (submissions.Submission, error)
}

type Judgements interface {
	Save(ctx context.Context, judgement judgements.Judgement) error
}

func New(
	clock cptime.Clock, db cpatomic.Transactioner, submitted Submissions, judged Judgements, outbox cpatomic.Outbox,
) *UseCase {
	return &UseCase{clock: clock, db: db, submissions: submitted, judgements: judged, outbox: outbox}
}

type UseCase struct {
	clock       cptime.Clock
	db          cpatomic.Transactioner
	submissions Submissions
	judgements  Judgements
	outbox      cpatomic.Outbox
}

// Keep keeps a judgement and what it tells, under the lock of its submission's row: a payment held at once sees the
// judgement, or is seen by it. False for a submission its player replaced meanwhile, which keeps none.
func (u *UseCase) Keep(ctx context.Context, judgement judgements.Judgement) (bool, error) {
	submission := judgement.TakeSnapshot().Submission
	kept := true
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		read, err := u.submissions.Submission(ctx, submission)
		if errors.Is(err, submissions.ErrUnknownSubmission) {
			kept = false
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read the submission: %w", err)
		}
		if err := u.judgements.Save(ctx, judgement); err != nil {
			return fmt.Errorf("failed to keep the judgement: %w", err)
		}
		if err := u.outbox.Record(ctx, judgement.Facts(read, u.clock.Now())...); err != nil {
			return fmt.Errorf("failed to record what the judgement tells: %w", err)
		}
		return nil
	}); err != nil {
		return false, fmt.Errorf("failed to keep the judgement of %s: %w", submission, err)
	}
	return kept, nil
}
