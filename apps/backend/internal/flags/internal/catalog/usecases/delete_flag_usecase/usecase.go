package delete_flag_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/pictures"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Flags interface {
	Flag(ctx context.Context, id catalog.FlagID) (catalog.Flag, error)
	Flags(ctx context.Context) ([]catalog.Flag, error)
	Delete(ctx context.Context, id catalog.FlagID) error
}

type Gallery interface {
	Unpublish(ctx context.Context, picture pictures.Snapshot) error
}

type In struct {
	Flag string
}

const timeout = 10 * time.Second

func New(
	clock cptime.Clock, db cpatomic.Transactioner, flags Flags, outbox cpatomic.Outbox, gallery Gallery,
) *UseCase {
	return &UseCase{clock: clock, db: db, flags: flags, outbox: outbox, gallery: gallery}
}

type UseCase struct {
	clock   cptime.Clock
	db      cpatomic.Transactioner
	flags   Flags
	outbox  cpatomic.Outbox
	gallery Gallery
}

func (u *UseCase) Execute(ctx context.Context, in In) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	id, err := catalog.ParseFlagID(in.Flag)
	if err != nil {
		return err //nolint:wrapcheck // the parse names it.
	}

	var gone catalog.Snapshot
	if err := u.db.InTx(ctx, func(ctx context.Context) error {
		flag, err := u.flags.Flag(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to read the flag: %w", err)
		}
		fact, err := flag.Deleted(u.clock.Now())
		if err != nil {
			return err //nolint:wrapcheck // the flag names it.
		}
		if err := u.flags.Delete(ctx, id); err != nil {
			return fmt.Errorf("failed to delete the flag: %w", err)
		}
		if err := u.outbox.Record(ctx, fact); err != nil {
			return fmt.Errorf("failed to record that the flag was deleted: %w", err)
		}
		gone = flag.TakeSnapshot()
		return nil
	}); err != nil {
		return fmt.Errorf("failed to delete the flag %s: %w", id, err)
	}

	// The flag is gone: a picture left public names nobody.
	rest, err := u.flags.Flags(ctx)
	if err == nil && !catalog.WornBy(rest, gone.Picture) {
		_ = u.gallery.Unpublish(ctx, gone.Picture)
	}
	return nil
}
