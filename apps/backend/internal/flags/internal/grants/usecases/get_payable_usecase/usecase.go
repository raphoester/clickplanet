package get_payable_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Flags interface {
	Flag(ctx context.Context, id catalog.FlagID) (catalog.Flag, error)
}

type In struct {
	Account string
	Target  grants.Target
}

func New(clock cptime.Clock, submissions submissions.Store, flags Flags) *UseCase {
	return &UseCase{clock: clock, submissions: submissions, flags: flags}
}

type UseCase struct {
	clock       cptime.Clock
	submissions submissions.Store
	flags       Flags
}

// Execute is nil when the account may buy time for the target now: its founder, while the time would keep the flag or
// bring it back.
func (u *UseCase) Execute(ctx context.Context, in In) error {
	account, err := submissions.ParseAccountID(in.Account)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	flagID, ok := in.Target.Flag()
	if !ok {
		id, _ := in.Target.Submission()
		submission, err := u.submissions.Submission(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to read the submission: %w", err)
		}
		if err := submission.PayableBy(account); err != nil {
			return err //nolint:wrapcheck // the submission names it.
		}
		if flagID, ok = submission.OpenedFlag(); !ok {
			return nil
		}
	} else {
		founded, err := u.submissions.Founded(ctx, account)
		if err != nil {
			return fmt.Errorf("failed to read the account's submissions: %w", err)
		}
		if err := submissions.FoundedBy(founded, flagID); err != nil {
			return err //nolint:wrapcheck // the rule names it.
		}
	}
	flag, err := u.flags.Flag(ctx, flagID)
	if err != nil {
		return fmt.Errorf("failed to read the flag: %w", err)
	}
	return flag.Payable(u.clock.Now()) //nolint:wrapcheck // the flag names it.
}
