package take_back_time_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Flags interface {
	Flag(ctx context.Context, id catalog.FlagID) (catalog.Flag, error)
	Save(ctx context.Context, flags ...catalog.Flag) error
}

func New(
	clock cptime.Clock, db cpatomic.Transactioner, grants grants.Store, flags Flags, outbox cpatomic.Outbox,
) *UseCase {
	return &UseCase{clock: clock, db: db, grants: grants, flags: flags, outbox: outbox}
}

type UseCase struct {
	clock  cptime.Clock
	db     cpatomic.Transactioner
	grants grants.Store
	flags  Flags
	outbox cpatomic.Outbox
}

// Execute takes a payment's time back once. Told before its grant, it is kept so the grant comes to nothing.
func (u *UseCase) Execute(ctx context.Context, id grants.GrantID) error {
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		grant, err := u.grants.Grant(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to read the grant: %w", err)
		}
		if grant.TakenBackAlready() {
			return nil
		}
		var flag catalog.Flag
		granted, ok := grant.GrantedFlag()
		if ok {
			if flag, err = u.flags.Flag(ctx, granted); err != nil {
				return fmt.Errorf("failed to read the flag: %w", err)
			}
		}
		taken, shortened, facts, err := grant.TakenBack(flag, u.clock.Now())
		if err != nil {
			return err //nolint:wrapcheck // the grant names it.
		}
		if ok {
			if err := u.flags.Save(ctx, shortened); err != nil {
				return fmt.Errorf("failed to keep the flag: %w", err)
			}
		}
		if err := u.grants.Save(ctx, taken); err != nil {
			return fmt.Errorf("failed to keep that the grant was taken back: %w", err)
		}
		if err := u.outbox.Record(ctx, facts...); err != nil {
			return fmt.Errorf("failed to record that the flag closed: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to take back the time of %s: %w", id, err)
	}
	return nil
}
