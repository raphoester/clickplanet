package appeal_submission_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/appeals"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type In struct {
	Account    string
	Linked     bool
	Submission string
}

// Out is the submission waiting again, and its appeal.
type Out struct {
	Submission submissions.Snapshot
	Appeal     appeals.Snapshot
}

type Judgements interface {
	Judgement(ctx context.Context, submission submissions.SubmissionID) (judgements.Judgement, error)
}

// Stores is what an appeal reads and changes: the submission and the account's others, the judge's judgement of it,
// and its appeals.
type Stores struct {
	Submissions submissions.Store
	Judgements  Judgements
	Appeals     appeals.Store
}

func New(clock cptime.Clock, db cpatomic.Transactioner, stores Stores, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{clock: clock, db: db, stores: stores, outbox: outbox}
}

type UseCase struct {
	clock  cptime.Clock
	db     cpatomic.Transactioner
	stores Stores
	outbox cpatomic.Outbox
}

// Execute asks a person to look again at the judge's rejection of the caller's submission: it waits again for the
// operator, and the operator is told.
func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	account, err := submissions.FounderOf(in.Account, in.Linked)
	if err != nil {
		return Out{}, err //nolint:wrapcheck // the founder names it.
	}
	id, err := submissions.ParseSubmissionID(in.Submission)
	if err != nil {
		return Out{}, err //nolint:wrapcheck // the parse names it.
	}
	var out Out
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		asking, err := u.asking(ctx, account, id)
		if err != nil {
			return err
		}
		appeal, submission, facts, err := appeals.Ask(asking, u.clock.Now())
		if err != nil {
			return err //nolint:wrapcheck // the appeal names it.
		}
		out = Out{Submission: submission.TakeSnapshot(), Appeal: appeal.TakeSnapshot()}
		if err := u.stores.Submissions.Save(ctx, submission); err != nil {
			return fmt.Errorf("failed to keep the submission: %w", err)
		}
		if err := u.stores.Appeals.Save(ctx, appeal); err != nil {
			return fmt.Errorf("failed to keep the appeal: %w", err)
		}
		if err := u.outbox.Record(ctx, facts...); err != nil {
			return fmt.Errorf("failed to record the appeal: %w", err)
		}
		return nil
	}); err != nil {
		return Out{}, err //nolint:wrapcheck // the transaction named it.
	}
	return out, nil
}

func (u *UseCase) asking(
	ctx context.Context, account catalog.AccountID, id submissions.SubmissionID,
) (appeals.Asking, error) {
	submission, err := u.stores.Submissions.Submission(ctx, id)
	if err != nil {
		return appeals.Asking{}, fmt.Errorf("failed to read the submission: %w", err)
	}
	founded, err := u.stores.Submissions.Founded(ctx, account)
	if err != nil {
		return appeals.Asking{}, fmt.Errorf("failed to read the account's submissions: %w", err)
	}
	judged, err := u.stores.Judgements.Judgement(ctx, id)
	if err != nil && !errors.Is(err, judgements.ErrUnjudged) {
		return appeals.Asking{}, fmt.Errorf("failed to read the judgement: %w", err)
	}
	earlier, err := u.stores.Appeals.Of(ctx, id)
	if err != nil {
		return appeals.Asking{}, fmt.Errorf("failed to read the submission's appeals: %w", err)
	}
	return appeals.Asking{
		Account: account, Submission: submission, Judgement: judged, Founded: founded, Earlier: earlier,
	}, nil
}
