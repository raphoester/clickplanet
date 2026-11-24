package open_submitted_flag_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

type Flags interface {
	Flags(ctx context.Context) ([]catalog.Flag, error)
	Save(ctx context.Context, flags ...catalog.Flag) error
}

type In struct {
	Submission string
	EndsAt     time.Time
}

func New(
	opening catalog.Opening, db cpatomic.Transactioner, store submissions.Store, flags Flags, outbox cpatomic.Outbox,
) *UseCase {
	return &UseCase{opening: opening, db: db, store: store, flags: flags, outbox: outbox}
}

type UseCase struct {
	opening catalog.Opening
	db      cpatomic.Transactioner
	store   submissions.Store
	flags   Flags
	outbox  cpatomic.Outbox
}

func (u *UseCase) Execute(ctx context.Context, in In) (catalog.Snapshot, error) {
	id, err := submissions.ParseSubmissionID(in.Submission)
	if err != nil {
		return catalog.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}
	approved, err := u.store.Submission(ctx, id)
	if err != nil {
		return catalog.Snapshot{}, fmt.Errorf("failed to read the submission: %w", err)
	}
	flags, err := u.flags.Flags(ctx)
	if err != nil {
		return catalog.Snapshot{}, fmt.Errorf("failed to read the flags' names: %w", err)
	}
	opened, flag, err := approved.Open(u.opening, catalog.NamesOf(flags), in.EndsAt)
	if err != nil {
		return catalog.Snapshot{}, err //nolint:wrapcheck // the rule names it.
	}

	// Never a flag the other modules are not told of, nor one opened twice from one submission.
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		if err := u.flags.Save(ctx, flag); err != nil {
			return fmt.Errorf("failed to keep the flag: %w", err)
		}
		if err := u.store.Save(ctx, opened); err != nil {
			return fmt.Errorf("failed to keep the submission opened: %w", err)
		}
		if err := u.outbox.Record(ctx, flag.Facts()...); err != nil {
			return fmt.Errorf("failed to record that the flag was opened: %w", err)
		}
		return nil
	}); err != nil {
		return catalog.Snapshot{}, fmt.Errorf("failed to open the flag of the submission %s: %w", id, err)
	}
	return flag.TakeSnapshot(), nil
}
