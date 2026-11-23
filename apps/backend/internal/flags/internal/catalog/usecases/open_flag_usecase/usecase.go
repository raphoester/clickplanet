//go:build testing

// Only tests call it until the operator's call to open and close a flag (step 2 of the regional flags) does.
package open_flag_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Flags interface {
	Save(ctx context.Context, flags ...catalog.Flag) error
}

type In struct {
	Name string
}

type Out struct {
	Flag catalog.FlagID
}

const timeout = 5 * time.Second

func New(ids catalog.IDProvider, clock cptime.Clock, db cpatomic.Transactioner, flags Flags, outbox cpatomic.Outbox) *UseCase {
	return &UseCase{ids: ids, clock: clock, db: db, flags: flags, outbox: outbox}
}

type UseCase struct {
	ids    catalog.IDProvider
	clock  cptime.Clock
	db     cpatomic.Transactioner
	flags  Flags
	outbox cpatomic.Outbox
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	name, err := catalog.ParseName(in.Name)
	if err != nil {
		return Out{}, err //nolint:wrapcheck // the parse names it.
	}
	id, err := u.ids.NewID()
	if err != nil {
		return Out{}, fmt.Errorf("failed to draw a flag id: %w", err)
	}
	flag := catalog.OpenFlag(id, name, u.clock.Now())

	// Never a flag the other modules are not told of, nor a fact of a flag not kept.
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		if err := u.flags.Save(ctx, flag); err != nil {
			return fmt.Errorf("failed to keep the flag: %w", err)
		}
		if err := u.outbox.Record(ctx, flag.Facts()...); err != nil {
			return fmt.Errorf("failed to record that the flag was opened: %w", err)
		}
		return nil
	}); err != nil {
		return Out{}, fmt.Errorf("failed to open the flag %q: %w", name, err)
	}
	return Out{Flag: id}, nil
}
