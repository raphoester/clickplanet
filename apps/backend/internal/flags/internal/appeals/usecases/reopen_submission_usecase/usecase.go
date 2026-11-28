package reopen_submission_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type In struct {
	Submission string
}

// Out is the submission waiting again, the judge's judgement of it, and the appeal it is kept as.
type Out struct {
	Submission submissions.Snapshot
	Judgement  judgements.Snapshot
	Appeal     appeals.Snapshot
}

type Judgements interface {
	Judgement(ctx context.Context, submission submissions.SubmissionID) (judgements.Judgement, error)
}

// Stores is what a reopen reads and changes: the submission and its founder's others, the judge's judgement of it,
// and its appeals.
type Stores struct {
	Submissions submissions.Store
	Judgements  Judgements
	Appeals     appeals.Store
}

func New(clock cptime.Clock, db cpatomic.Transactioner, stores Stores) *UseCase {
	return &UseCase{clock: clock, db: db, stores: stores}
}

type UseCase struct {
	clock  cptime.Clock
	db     cpatomic.Transactioner
	stores Stores
}

// Execute reopens the judge's rejection of a submission for the operator to decide, as its founder's appeal would.
func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	id, err := submissions.ParseSubmissionID(in.Submission)
	if err != nil {
		return Out{}, err //nolint:wrapcheck // the parse names it.
	}
	var out Out
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		reopening, err := u.reopening(ctx, id)
		if err != nil {
			return err
		}
		appeal, submission, err := appeals.Reopen(reopening, u.clock.Now())
		if err != nil {
			return err //nolint:wrapcheck // the appeal names it.
		}
		out = Out{
			Submission: submission.TakeSnapshot(), Judgement: reopening.Judgement.TakeSnapshot(),
			Appeal: appeal.TakeSnapshot(),
		}
		if err := u.stores.Submissions.Save(ctx, submission); err != nil {
			return fmt.Errorf("failed to keep the submission: %w", err)
		}
		if err := u.stores.Appeals.Save(ctx, appeal); err != nil {
			return fmt.Errorf("failed to keep the appeal: %w", err)
		}
		return nil
	}); err != nil {
		return Out{}, err //nolint:wrapcheck // the transaction named it.
	}
	return out, nil
}

func (u *UseCase) reopening(ctx context.Context, id submissions.SubmissionID) (appeals.Reopening, error) {
	submission, err := u.stores.Submissions.Submission(ctx, id)
	if err != nil {
		return appeals.Reopening{}, fmt.Errorf("failed to read the submission: %w", err)
	}
	founded, err := u.stores.Submissions.Founded(ctx, submission.TakeSnapshot().Account)
	if err != nil {
		return appeals.Reopening{}, fmt.Errorf("failed to read the founder's submissions: %w", err)
	}
	judged, err := u.stores.Judgements.Judgement(ctx, id)
	if err != nil && !errors.Is(err, judgements.ErrUnjudged) {
		return appeals.Reopening{}, fmt.Errorf("failed to read the judgement: %w", err)
	}
	earlier, err := u.stores.Appeals.Of(ctx, id)
	if err != nil {
		return appeals.Reopening{}, fmt.Errorf("failed to read the submission's appeals: %w", err)
	}
	return appeals.Reopening{Submission: submission, Judgement: judged, Founded: founded, Earlier: earlier}, nil
}
