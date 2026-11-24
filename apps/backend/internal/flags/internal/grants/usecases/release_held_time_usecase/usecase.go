package release_held_time_usecase

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/submissions"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Flags interface {
	Flag(ctx context.Context, id catalog.FlagID) (catalog.Flag, error)
	Flags(ctx context.Context) ([]catalog.Flag, error)
	Save(ctx context.Context, flags ...catalog.Flag) error
}

// Stores is what a release reads and changes: the time held, the submission it opens, and the flags.
type Stores struct {
	Grants      grants.Store
	Submissions submissions.Store
	Flags       Flags
}

func New(
	opening catalog.Opening, clock cptime.Clock, db cpatomic.Transactioner, stores Stores, outbox cpatomic.Outbox,
) *UseCase {
	return &UseCase{opening: opening, clock: clock, db: db, stores: stores, outbox: outbox}
}

type UseCase struct {
	opening catalog.Opening
	clock   cptime.Clock
	db      cpatomic.Transactioner
	stores  Stores
	outbox  cpatomic.Outbox
}

// Execute opens an approved submission's flag with the time paid while it waited, from now. Time released is held no
// more: told again, it releases nothing.
func (u *UseCase) Execute(ctx context.Context, submission string) error {
	id, err := submissions.ParseSubmissionID(submission)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		return u.release(ctx, id)
	}); err != nil {
		return fmt.Errorf("failed to release the time held for %s: %w", id, err)
	}
	return nil
}

func (u *UseCase) release(ctx context.Context, id submissions.SubmissionID) error {
	approved, err := u.stores.Submissions.Submission(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to read the submission: %w", err)
	}
	held, err := u.stores.Grants.Held(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to read the time held: %w", err)
	}
	if len(held) == 0 {
		return nil
	}
	flags, err := u.stores.Flags.Flags(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the flags' names: %w", err)
	}
	var flag catalog.Flag
	if opened, ok := approved.OpenedFlag(); ok {
		if flag, err = u.stores.Flags.Flag(ctx, opened); err != nil {
			return fmt.Errorf("failed to read the submission's flag: %w", err)
		}
	}
	var facts []proto.Message
	released := make([]grants.Grant, 0, len(held))
	for _, grant := range held {
		var granted grants.Grant
		var told []proto.Message
		granted, approved, flag, told, err = grant.Released(approved, flag, u.opening, catalog.NamesOf(flags),
			u.clock.Now())
		if err != nil {
			return err //nolint:wrapcheck // the grant names it.
		}
		released = append(released, granted)
		facts = append(facts, told...)
	}
	// The flag first: a grant names it.
	if err := u.stores.Flags.Save(ctx, flag); err != nil {
		return fmt.Errorf("failed to keep the flag: %w", err)
	}
	if err := u.stores.Submissions.Save(ctx, approved); err != nil {
		return fmt.Errorf("failed to keep the submission: %w", err)
	}
	for _, grant := range released {
		if err := u.stores.Grants.Save(ctx, grant); err != nil {
			return fmt.Errorf("failed to keep the grant: %w", err)
		}
	}
	if err := u.outbox.Record(ctx, facts...); err != nil {
		return fmt.Errorf("failed to record that the flag opened: %w", err)
	}
	return nil
}
