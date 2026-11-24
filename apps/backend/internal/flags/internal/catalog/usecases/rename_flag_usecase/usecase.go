package rename_flag_usecase

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

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
	Name string
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
	name, err := catalog.ParseName(in.Name)
	if err != nil {
		return catalog.Snapshot{}, err //nolint:wrapcheck // the parse names it.
	}

	var renamed catalog.Flag
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		flag, err := u.flags.Flag(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to read the flag: %w", err)
		}
		flags, err := u.flags.Flags(ctx)
		if err != nil {
			return fmt.Errorf("failed to read the flags' names: %w", err)
		}
		if err := catalog.NamesOf(flags).Free(name, id); err != nil {
			return err //nolint:wrapcheck // the names say which.
		}
		var fact proto.Message
		renamed, fact = flag.Renamed(name, u.clock.Now())
		if err := u.flags.Save(ctx, renamed); err != nil {
			return fmt.Errorf("failed to keep the flag: %w", err)
		}
		if err := u.outbox.Record(ctx, fact); err != nil {
			return fmt.Errorf("failed to record that the flag was renamed: %w", err)
		}
		return nil
	}); err != nil {
		return catalog.Snapshot{}, fmt.Errorf("failed to rename the flag %s: %w", id, err)
	}
	return renamed.TakeSnapshot(), nil
}
