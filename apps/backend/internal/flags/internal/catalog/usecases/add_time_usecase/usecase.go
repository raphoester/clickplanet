package add_time_usecase

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
	Flags(ctx context.Context) ([]catalog.Flag, error)
	Save(ctx context.Context, flags ...catalog.Flag) error
}

type In struct {
	Flag string
	Days uint32
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

// A flag that comes back is told opened again: the time added to an open one changes nothing the others know. A removed
// one comes back only while no new flag took its name.
func (u *UseCase) Execute(ctx context.Context, in In) (catalog.Snapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	id, err := catalog.ParseFlagID(in.Flag)
	if err != nil {
		return catalog.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}
	days, err := catalog.ParseDays(in.Days)
	if err != nil {
		return catalog.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}

	var added catalog.Flag
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		flag, err := u.flags.Flag(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to read the flag: %w", err)
		}
		flags, err := u.flags.Flags(ctx)
		if err != nil {
			return fmt.Errorf("failed to read the flags' names: %w", err)
		}
		added, err = flag.TimeAdded(days, catalog.NamesOf(flags), u.clock.Now())
		if err != nil {
			return err //nolint:wrapcheck // the flag names it.
		}
		if err := u.flags.Save(ctx, added); err != nil {
			return fmt.Errorf("failed to keep the flag: %w", err)
		}
		if flag.TakeSnapshot().State == added.TakeSnapshot().State {
			return nil
		}
		if err := u.outbox.Record(ctx, added.Facts()...); err != nil {
			return fmt.Errorf("failed to record that the flag came back: %w", err)
		}
		return nil
	}); err != nil {
		return catalog.Snapshot{}, fmt.Errorf("failed to add time to the flag %s: %w", id, err)
	}
	return added.TakeSnapshot(), nil
}
