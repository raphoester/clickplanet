package expire_flags_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Flags interface {
	Ending(ctx context.Context, at time.Time) ([]catalog.Flag, error)
	Save(ctx context.Context, flags ...catalog.Flag) error
}

const timeout = 30 * time.Second

func New(clock cptime.Clock, db cpatomic.Transactioner, flags Flags, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{clock: clock, db: db, flags: flags, outbox: outbox}
}

// UseCase closes every flag whose time ran out, at its end.
type UseCase struct {
	clock  cptime.Clock
	db     cpatomic.Transactioner
	flags  Flags
	outbox cpatomic.Outbox
}

func (u *UseCase) Execute(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	closed := 0
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		ending, err := u.flags.Ending(ctx, u.clock.Now())
		if err != nil {
			return fmt.Errorf("failed to read the flags whose time ran out: %w", err)
		}
		for _, flag := range ending {
			expired, ok := flag.Expired(u.clock.Now())
			if !ok {
				continue
			}
			if err := u.flags.Save(ctx, expired); err != nil {
				return fmt.Errorf("failed to keep the flag closed: %w", err)
			}
			if err := u.outbox.Record(ctx, expired.Facts()...); err != nil {
				return fmt.Errorf("failed to record that the flag closed: %w", err)
			}
			closed++
		}
		return nil
	}); err != nil {
		return 0, fmt.Errorf("failed to close the flags whose time ran out: %w", err)
	}
	return closed, nil
}
