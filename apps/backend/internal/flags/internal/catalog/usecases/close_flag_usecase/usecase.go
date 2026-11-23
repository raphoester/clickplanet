package close_flag_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Flags interface {
	Flag(ctx context.Context, id catalog.FlagID) (catalog.Flag, error)
	Save(ctx context.Context, flags ...catalog.Flag) error
}

type In struct {
	Flag string
}

const timeout = 5 * time.Second

func New(clock cptime.Clock, db cpatomic.Transactioner, flags Flags, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{clock: clock, db: db, flags: flags, outbox: outbox}
}

type UseCase struct {
	clock  cptime.Clock
	db     cpatomic.Transactioner
	flags  Flags
	outbox cpatomic.Outbox
}

func (u *UseCase) Execute(ctx context.Context, in In) (catalog.Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	id, err := catalog.ParseFlagID(in.Flag)
	if err != nil {
		return catalog.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}

	var closed catalog.Flag
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		flag, err := u.flags.Flag(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to read the flag: %w", err)
		}
		closed, err = flag.Closed(u.clock.Now())
		if err != nil {
			return err //nolint:wrapcheck // the flag names it.
		}
		if err := u.flags.Save(ctx, closed); err != nil {
			return fmt.Errorf("failed to keep the flag: %w", err)
		}
		if err := u.outbox.Record(ctx, closed.Facts()...); err != nil {
			return fmt.Errorf("failed to record that the flag was closed: %w", err)
		}
		return nil
	}); err != nil {
		return catalog.Snapshot{}, fmt.Errorf("failed to close the flag %s: %w", id, err)
	}
	return closed.TakeSnapshot(), nil
}
