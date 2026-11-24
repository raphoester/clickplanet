package judge_submissions_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/judgements"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Cases interface {
	Case(ctx context.Context, id submissions.SubmissionID) (judgements.Case, error)
}

type Judge interface {
	Judge(ctx context.Context, sent judgements.Case) (judgements.Ruling, error)
}

// Verdicts applies a ruling as the operator's would be: an approval opens a flag paid for, a rejection is refunded.
type Verdicts interface {
	Approve(ctx context.Context, id submissions.SubmissionID) error
	Reject(ctx context.Context, id submissions.SubmissionID, rejection submissions.Rejection) error
}

// The submissions of one round, one after the other.
const batch = 5

func New(clock cptime.Clock, store judgements.Store, cases Cases, judge Judge, verdicts Verdicts) *UseCase {
	return &UseCase{clock: clock, store: store, cases: cases, judge: judge, verdicts: verdicts}
}

type UseCase struct {
	clock    cptime.Clock
	store    judgements.Store
	cases    Cases
	judge    Judge
	verdicts Verdicts
}

// Execute judges the submissions that wait with no judgement, the first sent first, and applies each ruling. A
// submission is judged once: one the judge could not be asked of is asked again at the next round.
func (u *UseCase) Execute(ctx context.Context) (int, error) {
	unjudged, err := u.store.Unjudged(ctx, batch)
	if err != nil {
		return 0, fmt.Errorf("failed to list the submissions to judge: %w", err)
	}
	judged := 0
	for _, id := range unjudged {
		kept, err := u.judgeOne(ctx, id)
		if err != nil {
			return judged, fmt.Errorf("failed to judge %s: %w", id, err)
		}
		if kept {
			judged++
		}
	}
	return judged, nil
}

func (u *UseCase) judgeOne(ctx context.Context, id submissions.SubmissionID) (bool, error) {
	sent, err := u.cases.Case(ctx, id)
	if errors.Is(err, judgements.ErrNotWaiting) || errors.Is(err, submissions.ErrUnknownSubmission) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to read what the judge sees: %w", err)
	}
	ruling, err := u.judge.Judge(ctx, sent)
	if err != nil {
		return false, err //nolint:wrapcheck // the judge names it.
	}
	judgement, keep, err := judgements.Judged(id, ruling, u.clock.Now()).Applied(u.apply(ctx, id, ruling))
	if err != nil || !keep {
		return false, err //nolint:wrapcheck // the verdict names it.
	}
	if err := u.store.Save(ctx, judgement); err != nil {
		return false, fmt.Errorf("failed to keep the judgement: %w", err)
	}
	return true, nil
}

func (u *UseCase) apply(ctx context.Context, id submissions.SubmissionID, ruling judgements.Ruling) error {
	switch ruling.Verdict {
	case judgements.Approve:
		return u.verdicts.Approve(ctx, id) //nolint:wrapcheck // the verdict names it.
	case judgements.Reject:
		return u.verdicts.Reject(ctx, id, ruling.Rejection) //nolint:wrapcheck // the verdict names it.
	case judgements.Unsure:
	}
	return nil
}
