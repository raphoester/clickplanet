package grant_time_usecase

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

func New(
	opening catalog.Opening, clock cptime.Clock, db cpatomic.Transactioner, stores Stores, outbox cpatomic.Outbox,
) *UseCase {
	return &UseCase{opening: opening, clock: clock, db: db, stores: stores, outbox: outbox}
}

// Stores is what a grant reads and changes: the grant, the submission it may open, and the flags.
type Stores struct {
	Grants      grants.Store
	Submissions submissions.Store
	Flags       Flags
}

type UseCase struct {
	opening catalog.Opening
	clock   cptime.Clock
	db      cpatomic.Transactioner
	stores  Stores
	outbox  cpatomic.Outbox
}

// Execute grants a payment's time once: a payment told again, or taken back first, grants nothing. One the rules
// refuse is kept refused and told, so the shop refunds it: ErrRefused, with why.
func (u *UseCase) Execute(ctx context.Context, paid grants.Paid) error {
	var refusal error
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		grant, err := u.stores.Grants.Grant(ctx, paid.Grant)
		if err != nil {
			return fmt.Errorf("failed to read the grant: %w", err)
		}
		if grant.Heard() {
			return nil
		}
		granted, facts, err := u.grant(ctx, grant, paid)
		if grants.Refuses(err) {
			refusal = err
			refused, told := grant.Refused(u.clock.Now())
			granted, facts = refused, []proto.Message{told}
		} else if err != nil {
			return err
		}
		if err := u.stores.Grants.Save(ctx, granted); err != nil {
			return fmt.Errorf("failed to keep the grant: %w", err)
		}
		if err := u.outbox.Record(ctx, facts...); err != nil {
			return fmt.Errorf("failed to record what the grant told: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to grant the time of %s: %w", paid.Grant, err)
	}
	if refusal != nil {
		return fmt.Errorf("%w: %s: %w", grants.ErrRefused, paid.Grant, refusal)
	}
	return nil
}

func (u *UseCase) grant(ctx context.Context, grant grants.Grant, paid grants.Paid) (grants.Grant, []proto.Message, error) {
	now := u.clock.Now()
	if id, ok := paid.Target.Flag(); ok {
		flag, err := u.stores.Flags.Flag(ctx, id)
		if err != nil {
			return grant, nil, fmt.Errorf("failed to read the flag: %w", err)
		}
		granted, extended, facts, err := grant.ToFlag(flag, paid, now)
		if err != nil {
			return grant, nil, err //nolint:wrapcheck // the grant names it.
		}
		if err := u.stores.Flags.Save(ctx, extended); err != nil {
			return grant, nil, fmt.Errorf("failed to keep the flag: %w", err)
		}
		return granted, facts, nil
	}

	id, _ := paid.Target.Submission()
	submission, err := u.stores.Submissions.Submission(ctx, id)
	if err != nil {
		return grant, nil, fmt.Errorf("failed to read the submission: %w", err)
	}
	var flag catalog.Flag
	if opened, ok := submission.OpenedFlag(); ok {
		if flag, err = u.stores.Flags.Flag(ctx, opened); err != nil {
			return grant, nil, fmt.Errorf("failed to read the submission's flag: %w", err)
		}
	}
	flags, err := u.stores.Flags.Flags(ctx)
	if err != nil {
		return grant, nil, fmt.Errorf("failed to read the flags' names: %w", err)
	}
	granted, submission, flag, facts, err := grant.ToSubmission(
		submission, flag, u.opening, catalog.NamesOf(flags), paid, now)
	if err != nil {
		return grant, nil, err //nolint:wrapcheck // the grant names it.
	}
	if _, ok := granted.GrantedFlag(); !ok {
		return granted, facts, nil
	}
	if err := u.stores.Flags.Save(ctx, flag); err != nil {
		return grant, nil, fmt.Errorf("failed to keep the flag: %w", err)
	}
	if err := u.stores.Submissions.Save(ctx, submission); err != nil {
		return grant, nil, fmt.Errorf("failed to keep the submission: %w", err)
	}
	return granted, facts, nil
}
