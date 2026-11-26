package get_payable_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
)

type Flags interface {
	Flag(ctx context.Context, id catalog.FlagID) (catalog.Flag, error)
}

type In struct {
	Account string
	Target  grants.Target
}

func New(submissions submissions.Store, flags Flags) *UseCase {
	return &UseCase{submissions: submissions, flags: flags}
}

type UseCase struct {
	submissions submissions.Store
	flags       Flags
}

// Execute is every target whose time goes to the same flag, when the account may buy time for the target now: a
// submission's founder, a flag's holder, or anyone for a flag whose time ended.
func (u *UseCase) Execute(ctx context.Context, in In) ([]grants.Target, error) {
	account, err := catalog.ParseAccountID(in.Account)
	if err != nil {
		return nil, err //nolint:wrapcheck // the parse names it.
	}
	if id, ok := in.Target.Flag(); ok {
		if err := u.payableFlag(ctx, id, account); err != nil {
			return nil, err
		}
		opened, err := u.submissions.Opened(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to read the submission the flag opened from: %w", err)
		}
		targets := []grants.Target{in.Target}
		for _, submission := range opened {
			targets = append(targets, grants.SubmissionTarget(submission.TakeSnapshot().ID))
		}
		return targets, nil
	}

	id, _ := in.Target.Submission()
	submission, err := u.submissions.Submission(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to read the submission: %w", err)
	}
	if err := submission.PayableBy(account); err != nil {
		return nil, err //nolint:wrapcheck // the submission names it.
	}
	flag, ok := submission.OpenedFlag()
	if !ok {
		return []grants.Target{in.Target}, nil
	}
	if err := u.payableFlag(ctx, flag, account); err != nil {
		return nil, err
	}
	return []grants.Target{in.Target, grants.FlagTarget(flag)}, nil
}

func (u *UseCase) payableFlag(ctx context.Context, id catalog.FlagID, account catalog.AccountID) error {
	flag, err := u.flags.Flag(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to read the flag: %w", err)
	}
	return flag.PayableBy(account) //nolint:wrapcheck // the flag names it.
}
